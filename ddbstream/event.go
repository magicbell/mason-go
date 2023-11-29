package ddbstream

import (
	"context"
	"fmt"
)

// HandleFunc represents a function that can receive an event.
type HandleFunc func(context.Context, Event) error

// Event represents an event between core domains.
type Event struct {
	Source string
	Type   string
	ID     string
	PK     string
	SK     string
}

// String implements the Stringer interface.
func (e Event) String() string {
	return fmt.Sprintf(
		"ddbStream.Event{Source:%#v, Type:%#v, ID:%#v, PK:%#v, SK:%#v}",
		e.Source, e.Type, e.ID, e.PK, e.SK,
	)
}
