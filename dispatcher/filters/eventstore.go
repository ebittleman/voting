package filters

import (
	"github.com/ebittleman/voting/bus"
	"github.com/ebittleman/voting/eventstore"
)

// RefreshFilter calls refresh on an eventstore
type RefreshFilter struct {
	EventStore eventstore.EventStore
}

// Filter just calls refrest on an event store.
func (r *RefreshFilter) Filter(msg bus.Message) error {
	return r.EventStore.Refresh()
}
