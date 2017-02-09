package bus

import (
	"github.com/ebittleman/voting/eventmanager"
	"github.com/ebittleman/voting/eventstore"
)

type fowarder struct {
	bus          MessageQueue
	subs         []eventmanager.Subscription
	eventManager eventmanager.EventManager
	eventTypes   []string
}

// NewFowarder fowards all voting events to a message queue.
func NewFowarder(
	bus MessageQueue,
	eventTypes []string,
) eventmanager.Subscriber {
	f := new(fowarder)
	f.bus = bus
	f.eventTypes = eventTypes

	return f
}

// Subscribe binds to an event manager
func (f *fowarder) Subscribe(eventManager eventmanager.EventManager) {
	f.eventManager = eventManager

	for _, eventType := range f.eventTypes {
		f.subs = append(f.subs, eventManager.Subscribe(eventType, f.EventHandler))
	}
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
