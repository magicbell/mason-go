// Package ddbstream provides support for converting dynamodb stream events to core events.
package ddbstream

import (
	"context"
	"fmt"
	"strings"

	typesStream "github.com/aws/aws-sdk-go-v2/service/dynamodbstreams/types"
	"github.com/google/uuid"
	"github.com/magicbell-io/gofoundation/ddb"
	"go.uber.org/zap"
)

// Processor represents the ddbstream processor.
type Processor struct {
	log      *zap.SugaredLogger
	ddb      *ddb.Store
	handlers map[string]map[string][]HandleFunc
}

var mapping = map[typesStream.OperationType]string{
	typesStream.OperationTypeInsert: "created",
	typesStream.OperationTypeModify: "updated",
	typesStream.OperationTypeRemove: "deleted",
}

func NewProcessor(log *zap.SugaredLogger, ddb *ddb.Store) *Processor {
	return &Processor{
		log:      log,
		ddb:      ddb,
		handlers: map[string]map[string][]HandleFunc{},
	}
}

func (p *Processor) Process(ctx context.Context, records []*typesStream.Record) error {
	for _, record := range records {
		keys := record.Dynamodb.Keys
		pk := keys["PK"].(*typesStream.AttributeValueMemberS)
		sk := keys["SK"].(*typesStream.AttributeValueMemberS)

		parts := strings.Split(pk.Value, "#")
		if len(parts) != 2 {
			return fmt.Errorf("invalid pk: %s", pk.Value)
		}

		id, err := uuid.Parse(parts[1])
		if err != nil {
			return fmt.Errorf("ID is not UUID: %s", parts[1])
		}

		evt := Event{
			Source: parts[0],
			Type:   mapping[record.EventName],
			ID:     id,
			PK:     pk.Value,
			SK:     sk.Value,
		}
		p.log.Infow("ddbStream.Process", "event", evt)

		//   TODO: Only process records that aren't using composite keys and pk == sk?
		handlers, exist := p.handlers[evt.Source][evt.Type]
		if !exist {
			continue
		}

		for _, handler := range handlers {
			p.log.Infow("invoking registered handler", "event", evt)
			if err := handler(ctx, evt); err != nil {
				return err
			}
		}
	}

	return nil
}

// AddHandler adds a handler for a source and type.
func (p *Processor) RegisterHandler(source string, evtType string, fn HandleFunc) {
	handlers, exist := p.handlers[source]
	if !exist {
		handlers = map[string][]HandleFunc{}
	}
	p.handlers[source] = handlers

	handlers[evtType] = append(handlers[evtType], fn)
}
