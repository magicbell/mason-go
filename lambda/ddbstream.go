// Package lambda provides support for running the app inside a Lambda function.
package lambda

import (
	"context"

	"github.com/aws/aws-lambda-go/events"
	typesStream "github.com/aws/aws-sdk-go-v2/service/dynamodbstreams/types"
)

type Processor interface {
	Process(ctx context.Context, records []*typesStream.Record) error
}

// DDBStream represents the lambda DDBStream handler.
type DDBStream struct {
	Processor Processor
}

var opMapping = map[string]typesStream.OperationType{
	"INSERT": typesStream.OperationTypeInsert,
	"MODIFY": typesStream.OperationTypeModify,
	"REMOVE": typesStream.OperationTypeRemove,
}

// Handler processes the DynamoDB event.
func (d *DDBStream) Handler(ctx context.Context, evt events.DynamoDBEvent) error {
	records := make([]*typesStream.Record, len(evt.Records))
	for i, record := range evt.Records {
		change := record.Change
		records[i] = &typesStream.Record{
			AwsRegion: &record.AWSRegion,
			Dynamodb: &typesStream.StreamRecord{
				ApproximateCreationDateTime: &change.ApproximateCreationDateTime.Time,
				Keys: map[string]typesStream.AttributeValue{
					"PK": &typesStream.AttributeValueMemberS{Value: change.Keys["PK"].String()},
					"SK": &typesStream.AttributeValueMemberS{Value: change.Keys["SK"].String()},
				},
				NewImage:       map[string]typesStream.AttributeValue{},
				OldImage:       map[string]typesStream.AttributeValue{},
				SequenceNumber: &change.SequenceNumber,
				SizeBytes:      &change.SizeBytes,
				StreamViewType: typesStream.StreamViewTypeKeysOnly,
			},
			EventID:      &record.EventID,
			EventName:    opMapping[record.EventName],
			EventSource:  &record.EventSource,
			EventVersion: &record.EventVersion,
		}
	}

	return d.Processor.Process(ctx, records)
}
