package eventstore

// Event an event store record
type Event struct {
	ID        string `json:"id"`
	Version   int64  `json:"version"`
	Type      string `json:"type"`
	Timestamp int64  `json:"timestamp"`
	Data      []byte `json:"data"`
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
	return e[i].ID < e[j].ID && e[i].Version < e[j].Version
}

// EventStore component that manages system events.
type EventStore interface {
	Query(string) (Events, error)
	QueryByEventType(string) (Events, error)
	Put(string, int64, Event) error
}
