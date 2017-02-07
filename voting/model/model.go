package model

import (
	"log"
	"time"

	"github.com/ebittleman/voting/eventstore"
)

// AggregateRoot entity event streams are partitioned on
type AggregateRoot struct {
	ID      string
	Version int64

	events eventstore.Events
}

// Emit adds an event to the current transaction
func (a *AggregateRoot) Emit(event eventstore.Event) {
	var version int64
	if numEvents := len(a.events); numEvents > 0 {
		version = a.events[numEvents-1].Version + 1
	} else {
		version = a.Version + 1
	}

	event.ID = a.ID
	event.Version = version
	event.Timestamp = time.Now().UTC().Unix()

	a.events = append(a.events, event)
}

// Flush prepares the entity for committing a transaction.
func (a *AggregateRoot) Flush() eventstore.Events {
	events := a.events
	a.events = nil
	return events
}

// Commit writes open events to an event store. TODO: look at inverting this
// relationship.
func (a *AggregateRoot) Commit(store eventstore.EventStore, events eventstore.Events) error {
	version := a.Version
	for _, event := range events {
		if err := store.Put(a.ID, version, event); err != nil {
			log.Println("Current Version: ", version, " Event Version: ", event.Version)
			return err
		}

		version = event.Version
	}

	a.Version = version

	return nil
}
