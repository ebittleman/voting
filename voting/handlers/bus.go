package handlers

import (
	"io"

	"github.com/ebittleman/voting/bus"
	"github.com/ebittleman/voting/eventmanager"
	"github.com/ebittleman/voting/eventstore"
)

type fowarder struct {
	bus          bus.MessageQueue
	subs         []eventmanager.Subscription
	eventManager eventmanager.EventManager
}

// NewFowarder fowards all voting events to a message queue.
func NewFowarder(
	bus bus.MessageQueue,
	eventManager eventmanager.EventManager,
) io.Closer {
	f := new(fowarder)
	f.bus = bus
	f.eventManager = eventManager

	for _, eventType := range eventTypes {
		f.subs = append(f.subs, eventManager.Subscribe(eventType, f.EventHandler))
	}

	return f
}

// EventHandler routes events from events published by an event manager
func (f *fowarder) EventHandler(event eventstore.Event) error {
	return f.bus.Send(event)
}

func (f *fowarder) Close() error {
	for _, sub := range f.subs {
		if err := f.eventManager.Unsubscribe(sub); err != nil {
			return err
		}
	}

	return nil
}
