package json

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"sync"

	jsondb "code.bittles.net/voting/database/json"
	"code.bittles.net/voting/eventstore"
)

// New creates a json backed event store
func New(conn *jsondb.Connection) eventstore.EventStore {
	table := new(table)
	conn.RegisterTable("events", table)

	store := new(store)
	store.table = table

	return store
}

type store struct {
	table jsondb.Table
	sync.RWMutex
}

func (s *store) Query(id string) (eventstore.Events, error) {
	s.RLock()
	defer s.RUnlock()
	var events eventstore.Events

	for records := range s.table.Scan() {
		event := new(eventstore.Event)
		json.Unmarshal(records, event)
		if event.ID != id {
			continue
		}

		events = append(events, *event)
	}

	sort.Sort(events)

	return events, nil
}

func (s *store) Put(id string, version int64, event eventstore.Event) error {
	if event.Version == version {
		return errors.New("Conflict Error")
	}

	events, err := s.Query(id)
	if err != nil {
		return err
	}

	s.Lock()
	defer s.Unlock()
	if num := len(events); num > 0 && events[num-1].Version != version {
		return fmt.Errorf("Conflict Error")
	}

	return s.table.Put(event)
}

type table struct {
	records []json.RawMessage
}

func (t *table) Scan() chan json.RawMessage {

	records := make(chan json.RawMessage)
	go func() {
		for _, record := range t.records {
			records <- record
		}
		close(records)
	}()

	return records
}

func (t *table) Put(v interface{}) error {
	var (
		record []byte
		err    error
	)

	if record, err = json.Marshal(v); err != nil {
		return err
	}

	t.records = append(t.records, record)

	return nil
}

func (t *table) Load(records chan json.RawMessage) {
	for record := range records {
		t.records = append(t.records, record)
	}
}
