package json

import (
	"encoding/json"
	"log"
	"sync"

	jsondb "github.com/ebittleman/voting/database/json"
	"github.com/ebittleman/voting/views"
)

type store struct {
	table *table
}

// NewStore initializes a new jsondb backed ViewStore
func NewStore(conn *jsondb.Connection) (views.ViewStore, error) {
	table := new(table)
	table.records = make(map[string]views.ViewRow)
	if err := conn.RegisterTable("views", table); err != nil {
		return nil, err
	}

	store := new(store)
	store.table = table

	return store, nil
}

func (s *store) Get(id string) (row views.ViewRow, err error) {
	row, ok := s.table.records[id]
	if !ok {
		return row, views.ErrRowNotFound
	}

	return
}

func (s *store) Put(row views.ViewRow) error {
	return s.table.Put(row)
}

type table struct {
	records map[string]views.ViewRow
	sync.RWMutex
}

func (t *table) Scan() chan json.RawMessage {
	records := make(chan json.RawMessage)
	go func() {
		t.RLock()
		defer t.RUnlock()
		defer close(records)
		var (
			record []byte
			err    error
		)
		for _, viewRow := range t.records {
			if record, err = json.Marshal(&viewRow); err != nil {
				log.Println("Error: Marshaling views.ViewRow: ", err)
				return
			}
			records <- record
		}
	}()

	return records
}

func (t *table) Put(v interface{}) error {
	t.Lock()
	defer t.Unlock()

	var (
		row views.ViewRow
		ok  bool
	)

	if row, ok = v.(views.ViewRow); !ok {
		return views.ErrExpectedViewRow
	}

	t.records[row.ID] = row

	return nil
}

func (t *table) Load(records chan json.RawMessage) error {
	t.Lock()
	defer t.Unlock()

	for record := range records {
		row := new(views.ViewRow)
		if err := json.Unmarshal(record, row); err != nil {
			return err
		}
		t.records[row.ID] = *row
	}

	return nil
}
