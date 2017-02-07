package views

import (
	"encoding/json"
	"errors"
	"log"

	jsondb "github.com/ebittleman/voting/database/json"
)

// ErrExpectedViewRow returned when an invalid type is passed to a views table
var ErrExpectedViewRow = errors.New("Expected views.ViewRow")

// ViewRow holds view data
type ViewRow struct {
	ID   string           `json:"id"`
	Data *json.RawMessage `json:"data"`
}

type table struct {
	records map[string]ViewRow
}

// NewTable registers a new jsondb table for placing vieViewRows into.
func NewTable(conn *jsondb.Connection) (jsondb.Table, error) {
	table := new(table)
	table.records = make(map[string]ViewRow)
	if err := conn.RegisterTable("views", table); err != nil {
		return nil, err
	}

	return table, nil
}

func (t *table) Scan() chan json.RawMessage {
	records := make(chan json.RawMessage)
	go func() {
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
	var (
		row ViewRow
		ok  bool
	)

	if row, ok = v.(ViewRow); !ok {
		return ErrExpectedViewRow
	}

	t.records[row.ID] = row

	return nil
}

func (t *table) Load(records chan json.RawMessage) error {
	for record := range records {
		row := new(ViewRow)
		if err := json.Unmarshal(record, row); err != nil {
			return err
		}
		t.records[row.ID] = *row
	}

	return nil
}
