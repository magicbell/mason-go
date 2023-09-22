package ddbstream

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

// HandleFunc represents a function that can receive an event.
type HandleFunc func(context.Context, Event) error

// Event represents an event between core domains.
type Event struct {
	Source string
	Type   string
	ID     uuid.UUID
	PK     string
	SK     string
}

// String implements the Stringer interface.
func (e Event) String() string {
	return fmt.Sprintf(
		"ddbStream.Event{Source:%#v, Type:%#v, ID:%#v, PK:%#v, SK:%#v}",
		e.Source, e.Type, e.ID.String(), e.PK, e.SK,
	)
}
