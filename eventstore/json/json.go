package json

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"sync"

	jsondb "github.com/ebittleman/voting/database/json"
	"github.com/ebittleman/voting/eventstore"
)

const tableName = "events"

// New creates a json backed event store
func New(conn *jsondb.Connection) (eventstore.EventStore, error) {
	table := new(table)
	if err := conn.RegisterTable(tableName, table); err != nil {
		return nil, err
	}

	store := new(store)
	store.conn = conn
	store.table = table

	return store, nil
}

type store struct {
	conn  *jsondb.Connection
	table *table
}

func (s *store) Refresh() error {
	s.conn.UnregisterTable(tableName)
	s.table.reset()

	if err := s.conn.RegisterTable(tableName, s.table); err != nil {
		return err
	}

	return nil
}

func (s *store) QueryByEventType(eventType string) (eventstore.Events, error) {
	var events eventstore.Events

	for records := range s.table.Scan() {
		event := new(eventstore.Event)
		if err := json.Unmarshal(records, event); err != nil {
			return nil, err
		}

		if event.Type != eventType {
			continue
		}

		events = append(events, *event)
	}

	sort.Sort(events)

	return events, nil
}

func (s *store) Query(id string) (eventstore.Events, error) {
	var events eventstore.Events

	for records := range s.table.Scan() {
		event := new(eventstore.Event)
		if err := json.Unmarshal(records, event); err != nil {
			return nil, err
		}

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

	if num := len(events); num > 0 &&
		(events[num-1].Version != version || event.Version < version) {
		return fmt.Errorf("Conflict Error")
	}

	return s.table.Put(event)
}

func (s *store) Snapshot(event eventstore.Event, snapshot interface{}) error {
	return nil
}

type table struct {
	records []json.RawMessage
	sync.RWMutex
}

func (t *table) Scan() chan json.RawMessage {

	records := make(chan json.RawMessage)
	go func() {
		t.RLock()
		defer t.RUnlock()
		for _, record := range t.records {
			records <- record
		}
		close(records)
	}()

	return records
}

func (t *table) Put(v interface{}) error {
	t.Lock()
	defer t.Unlock()

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

func (t *table) Load(records chan json.RawMessage) error {
	t.Lock()
	defer t.Unlock()

	for record := range records {
		t.records = append(t.records, record)
	}

	return nil
}

func (t *table) reset() {
	t.Lock()
	defer t.Unlock()
	t.records = nil
}
