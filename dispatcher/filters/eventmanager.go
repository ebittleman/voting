package filters

import (
	"github.com/ebittleman/voting/bus"
	"github.com/ebittleman/voting/dispatcher"
)

// EventTypeFilter returns nil for allowed events. Returns dispatcher.ErrNack
// for all others.
type EventTypeFilter struct {
	AllowedEventTypes []string
}

// Filter returns nil for allowed events. Returns dispatcher.ErrNack
// for all others.
func (e EventTypeFilter) Filter(msg bus.Message) error {
	for _, eventType := range e.AllowedEventTypes {
		if eventType == msg.Event().Type {
			return nil
		}
	}

	return dispatcher.ErrNack
}
