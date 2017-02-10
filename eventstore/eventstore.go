package eventstore

import "encoding/json"

// Event an event store record
type Event struct {
	ID        string           `json:"id"`
	Version   int64            `json:"version"`
	Type      string           `json:"type"`
	Timestamp int64            `json:"timestamp"`
	Data      *json.RawMessage `json:"data,omitempty"`
}

// Events list of Event
type Events []Event

func (e Events) Len() int {
	return len(e)
}
func (e Events) Swap(i, j int) {
	e[i], e[j] = e[j], e[i]
}
func (e Events) Less(i, j int) bool {
	if e[i].ID == e[j].ID {
		return e[i].Version < e[j].Version
	}

	return e[i].ID < e[j].ID
}

// EventStore component that manages system events.
type EventStore interface {
	Query(string) (Events, error)
	QueryByEventType(string) (Events, error)
	Put(string, int64, Event) error
	Refresh() error
}
