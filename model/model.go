package model

import (
	"log"
	"time"

	"code.bittles.net/voting/eventstore"
)

// AggregateRoot entity event streams are partitioned on
type AggregateRoot struct {
	ID      string
	version int64

	events eventstore.Events
}

// Emit adds an event to the current transaction
func (a *AggregateRoot) Emit(event eventstore.Event) {
	var version int64
	if numEvents := len(a.events); numEvents > 0 {
		version = a.events[numEvents-1].Version + 1
	} else {
		version = a.version + 1
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
func (a *AggregateRoot) Commit(store eventstore.EventStore) error {
	version := a.version
	events := a.Flush()
	for _, event := range events {
		if err := store.Put(a.ID, version, event); err != nil {
			log.Println("Current Version: ", version, " Event Version: ", event.Version)
			return err
		}

		version = event.Version
	}

	a.version = version

	return nil
}
