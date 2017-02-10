package couchdb

import (
	"fmt"
	"sort"

	"github.com/ebittleman/voting/eventstore"
	couchdb "github.com/fjl/go-couchdb"
)

const dbName = "voting-1486696777"

var emptyObject = map[string]interface{}{}

// New initializes a new couchdb backed eventstore
func New(client *couchdb.Client) (eventstore.EventStore, error) {
	if err := client.Ping(); err != nil {
		return nil, err
	}

	db := client.DB(dbName)
	return &store{
		client: client,
		db:     db,
	}, nil
}

type store struct {
	client *couchdb.Client
	db     *couchdb.DB
}

func (s *store) Refresh() error {
	return nil
}

func (s *store) QueryByEventType(eventType string) (eventstore.Events, error) {
	var events eventstore.Events
	page := new(viewPage)
	if err := s.db.View("_design/indexes", "by_type", page, couchdb.Options{
		"reduce":       false,
		"include_docs": true,
		"limit":        20,
		"startkey":     []interface{}{eventType},
		"endkey":       []interface{}{eventType + "z", emptyObject, emptyObject},
	}); err != nil {
		return nil, err
	}

	for _, row := range page.Rows {
		if row.Doc == nil {
			continue
		}

		events = append(events, *row.Doc)
	}

	sort.Sort(events)
	return events, nil
}

func (s *store) Query(id string) (eventstore.Events, error) {
	var events eventstore.Events
	page := new(viewPage)
	if err := s.db.View("_design/indexes", "events", page, couchdb.Options{
		"reduce":       false,
		"include_docs": true,
		"descending":   true,
		"limit":        20,
		"endkey":       []interface{}{id},
		"startkey":     []interface{}{id, emptyObject},
	}); err != nil {
		return nil, err
	}

	for _, row := range page.Rows {
		if row.Doc == nil {
			continue
		}

		events = append(events, *row.Doc)
	}

	sort.Sort(events)
	return events, nil
}

func (s *store) Put(id string, version int64, event eventstore.Event) error {
	docID := fmt.Sprintf("%s-%d", id, event.Version)
	_, err := s.db.Put(docID, event, "")
	return err
}

type viewPage struct {
	TotalRows int   `json:"total_rows"`
	Offset    int   `json:"offset"`
	Rows      []row `json:"rows"`
}

type row struct {
	ID  string            `json:"id"`
	Key []interface{}     `json:"key"`
	Doc *eventstore.Event `json:"doc,omitempty"`
}
