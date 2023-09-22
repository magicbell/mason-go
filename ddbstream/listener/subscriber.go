// package listener offers a way to subscribe to a DynamoDB stream without lambda triggers.
package listener

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodbstreams"
	"github.com/aws/aws-sdk-go-v2/service/dynamodbstreams/types"
	"golang.org/x/sync/errgroup"
)

type Stream struct {
	api       *dynamodb.Client
	streamAPI *dynamodbstreams.Client
	options   Options
	tableName string
}

type idSet struct {
	mutex sync.Mutex
	data  map[string]time.Time
}

func (m *idSet) AddAll(ss ...*types.Shard) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	for _, s := range ss {
		if m.data == nil {
			m.data = map[string]time.Time{}
		}

		id := aws.ToString(s.ShardId)
		m.data[id] = time.Now()
	}
}

func (m *idSet) Contains(id string) bool {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	_, ok := m.data[id]
	return ok
}

func (m *idSet) Expire(age time.Duration) int {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	n := 0
	cutoff := time.Now().Add(-age)
	for k, v := range m.data {
		if v.Before(cutoff) {
			delete(m.data, k)
			n++
		}
	}

	return n
}

func (m *idSet) Remove(id string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	delete(m.data, id)
}

func (m *idSet) Size() int {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	return len(m.data)
}

func (m *idSet) Slice() (ss []string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	for v := range m.data {
		ss = append(ss, v)
	}
	return ss
}

type Subscriber struct {
	stream  *Stream
	cancel  context.CancelFunc
	done    chan struct{}
	err     error
	invoker invokeFunc
	options Options
}

func New(api *dynamodb.Client, streamAPI *dynamodbstreams.Client, tableName *string, opts ...Option) *Stream {
	options := buildOptions(opts...)
	return &Stream{
		api:       api,
		options:   options,
		streamAPI: streamAPI,
		tableName: *tableName,
	}
}

func (s *Stream) Subscribe(ctx context.Context, v interface{}) (*Subscriber, error) {
	ctx, cancel := context.WithCancel(ctx)

	describeInput := dynamodb.DescribeTableInput{
		TableName: aws.String(s.tableName),
	}
	describeOutput, err := s.api.DescribeTable(ctx, &describeInput)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("unable to subscribe to table, %v: %w", s.tableName, err)
	}
	if table := describeOutput.Table; table == nil || table.StreamSpecification == nil {
		cancel()
		return nil, fmt.Errorf("unable to subscribe to table, %v: no stream specification found", s.tableName)
	}
	streamARN := aws.ToString(describeOutput.Table.LatestStreamArn)

	subscriber := &Subscriber{
		stream:  s,
		cancel:  cancel,
		done:    make(chan struct{}),
		invoker: newInvoker(streamARN, v),
		options: s.options,
	}

	go func() {
		defer close(subscriber.done)
		defer cancel()
		subscriber.err = subscriber.mainLoop(ctx, streamARN)
	}()

	return subscriber, nil
}

func (s *Subscriber) getShards(ctx context.Context, streamARN string) ([]*types.Shard, error) {
	var (
		shards       []*types.Shard
		startShardID *string
	)

	for {
		input := dynamodbstreams.DescribeStreamInput{
			ExclusiveStartShardId: startShardID,
			Limit:                 aws.Int32(100),
			StreamArn:             aws.String(streamARN),
		}
		output, err := s.stream.streamAPI.DescribeStream(ctx, &input)
		if err != nil {
			return nil, fmt.Errorf("unable to describe dynamodb stream, %v: %w", streamARN, err)
		}

		s.options.debug("found %v shards", len(output.StreamDescription.Shards))
		for _, shard := range output.StreamDescription.Shards {
			shards = append(shards, &shard)
		}

		startShardID = output.StreamDescription.LastEvaluatedShardId
		if startShardID == nil {
			break
		}
	}

	return shards, nil
}

func (s *Subscriber) mainLoop(ctx context.Context, streamARN string) (err error) {
	var (
		ch        = make(chan *types.Record)
		next      = make(chan struct{}, 1)
		wip       = &idSet{}
		completed = &idSet{}
	)

	group, ctx := errgroup.WithContext(ctx)
	group.Go(func() error {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		expire := time.NewTicker(12 * time.Hour)
		defer expire.Stop()

		for {
			s.options.debug("scanning for shards")

			shards, err := s.getShards(ctx, streamARN)
			if err != nil {
				return err
			}

			excludeCompleted := func(shard *types.Shard) bool {
				return !containsString(completed.Slice(), aws.ToString(shard.ShardId))
			}
			shards = filter(shards, excludeCompleted) // ignore any shards we've completed

			d := dag{}
			d.addShards(shards...)

			for _, item := range d.Roots() {
				shard := item
				shardID := aws.ToString(shard.ShardId)
				if wip.Contains(shardID) {
					continue
				}

				wip.AddAll(shard)
				go func() {
					defer wip.Remove(shardID)
					defer completed.AddAll(shard)

					select {
					case next <- struct{}{}:
					default:
					}

					s.options.debug("shard started, %v\n", shardID)
					defer s.options.debug("shard completed, %v\n", shardID)

					err := s.iterateShardWithRetry(ctx, streamARN, shard, ch)
					if err != nil {
						s.options.debug("iterate shard failed, %v", err)
					}
				}()
			}

			select {
			case <-ctx.Done():
				return nil
			case <-next:
				continue

			case <-expire.C:
				n := completed.Expire(36 * time.Hour)
				s.options.debug("expired %v elements", n)

			case <-ticker.C:
				continue
			}
		}
	})
	group.Go(func() error {
		ticker := time.NewTicker(s.options.pollInterval)
		defer ticker.Stop()

		var records []*types.Record
		invoke := func() error {
			if len(records) == 0 {
				return nil
			}

			for {
				if err := s.invoker(ctx, records); err != nil {
					delay := 3 * time.Second
					fmt.Printf("callback failed - %v; will retry in %v\n", err, delay)

					select {
					case <-ctx.Done():
						return ctx.Err()
					case <-time.After(delay):
						continue
					}
				}
				break
			}

			records = nil
			return nil
		}

		for {
			select {
			case <-ctx.Done():
				return nil

			case v := <-ch:
				records = append(records, v)
				if len(records) >= s.stream.options.batchSize {
					if err := invoke(); err != nil {
						return err
					}
				}

			case <-ticker.C:
				if err := invoke(); err != nil {
					return err
				}
			}
		}
	})
	return group.Wait()
}

func filter(shards []*types.Shard, conditions ...func(shard *types.Shard) bool) (ss []*types.Shard) {
loop:
	for _, shard := range shards {
		for _, fn := range conditions {
			if !fn(shard) {
				continue loop
			}
		}
		ss = append(ss, shard)
	}
	return ss
}

func (s *Subscriber) iterateShardWithRetry(ctx context.Context, streamARN string, shard *types.Shard, ch chan *types.Record) error {
	var sequenceNumber *string
	var multiplier time.Duration = 1
	for {
		if multiplier > 16 {
			multiplier = 16
		}

		if err := s.iterateShard(ctx, streamARN, shard, ch, sequenceNumber); err != nil {
			var ae *types.LimitExceededException
			if errors.As(err, &ae) {
				multiplier *= 2
				delay := time.Second * multiplier
				s.options.debug("rate limit exceeded, pausing %v", delay)

				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(delay):
					continue
				}
			}
			return err
		}
		return nil
	}
}

func (s *Subscriber) iterateShard(ctx context.Context, streamARN string, shard *types.Shard, ch chan *types.Record, sequenceNumber *string) error {
	iteratorType := s.options.shardIteratorType
	if sequenceNumber != nil {
		iteratorType = string(types.ShardIteratorTypeAfterSequenceNumber)
	}

	for {
		iterInput := dynamodbstreams.GetShardIteratorInput{
			SequenceNumber:    sequenceNumber,
			ShardId:           shard.ShardId,
			ShardIteratorType: types.ShardIteratorType(iteratorType),
			StreamArn:         aws.String(streamARN),
		}
		iterOutput, err := s.stream.streamAPI.GetShardIterator(ctx, &iterInput)
		if err != nil {
			return fmt.Errorf("failed to retrieve iterator for shard, %v: %w", aws.ToString(shard.ShardId), err)
		}

		iterator := iterOutput.ShardIterator
		for {
			input := dynamodbstreams.GetRecordsInput{
				ShardIterator: iterator,
				Limit:         aws.Int32(1000),
			}
			output, err := s.stream.streamAPI.GetRecords(ctx, &input)
			if err != nil {
				return fmt.Errorf("failed to get records from shard, %v: %w", aws.ToString(iterOutput.ShardIterator), err)
			}

			for _, record := range output.Records {
				select {
				case <-ctx.Done():
					return nil
				case ch <- &record:
					// ok
				}
			}

			iterator = output.NextShardIterator
			if iterator == nil {
				return nil
			}

			if len(output.Records) == 0 {
				select {
				case <-ctx.Done():
					return nil
				case <-time.After(3 * time.Second):
					// ok
				}
			}
		}
	}
}

func (s *Subscriber) Close() error {
	s.cancel()
	<-s.done
	return s.err
}
