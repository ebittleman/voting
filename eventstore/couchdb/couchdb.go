package couchdb

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"sort"

	"github.com/ebittleman/voting/eventstore"
	couchdb "github.com/fjl/go-couchdb"
)

const (
	dbName   = "events"
	pageSize = 250
)

var emptyObject = map[string]interface{}{}

type attachment struct {
	Stub        bool   `json:"stub"`
	ContentType string `json:"content_type"`
	Length      int64  `json:"length"`
}

type docWrapper struct {
	ID          string                `json:"_id"`
	Rev         string                `json:"_rev"`
	Attachments map[string]attachment `json:"_attachments,omitempty"`
	eventstore.Event
}

type row struct {
	ID      string        `json:"id"`
	Key     []interface{} `json:"key"`
	Wrapper *docWrapper   `json:"doc,omitempty"`
}

type viewPage struct {
	TotalRows int   `json:"total_rows"`
	Offset    int   `json:"offset"`
	Rows      []row `json:"rows"`
}

type paginateViewInput struct {
	DesignDoc string
	View      string
	Options   couchdb.Options
}

type paginateViewCallback func(*viewPage, bool) bool

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

func (s *store) QueryByEventType(
	eventType string,
) (events eventstore.Events, err error) {
	input := &paginateViewInput{
		DesignDoc: "_design/indexes",
		View:      "by_type",
		Options: couchdb.Options{
			"reduce":        false,
			"include_docs":  true,
			"limit":         pageSize,
			"inclusive_end": true,
			"start_key":     []interface{}{eventType},
			"end_key":       []interface{}{eventType, emptyObject, emptyObject},
		},
	}

	if err = paginateView(s.db, input, func(page *viewPage, lastPage bool) bool {
		for _, row := range page.Rows {
			if row.Wrapper == nil {
				log.Println("No Doc")
				continue
			}

			events = append(events, row.Wrapper.Event)
		}
		return true
	}); err != nil {
		return nil, err
	}

	sort.Sort(events)
	return events, nil
}

func (s *store) Query(
	id string,
) (events eventstore.Events, err error) {
	input := &paginateViewInput{
		DesignDoc: "_design/indexes",
		View:      "events",
		Options: couchdb.Options{
			"reduce":        false,
			"include_docs":  true,
			"descending":    true,
			"limit":         pageSize,
			"inclusive_end": true,
			"start_key":     []interface{}{id, emptyObject},
			"end_key":       []interface{}{id},
		},
	}

	if err = paginateView(s.db, input, func(page *viewPage, lastPage bool) bool {
		for _, row := range page.Rows {
			if row.Wrapper == nil {
				continue
			}

			done, resolveErr := resolveSnapshot(s.db, row.Wrapper)
			if resolveErr != nil {
				fmt.Println("Error: Error Resolving Snapshot: ", resolveErr)
				done = false
			}

			events = append(events, row.Wrapper.Event)

			if done {
				return false
			}
		}
		return true
	}); err != nil {
		return nil, err
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

	docID := fmt.Sprintf("%s-%d", id, event.Version)
	_, err = s.db.Put(docID, event, "")
	return err
}

func (s *store) Snapshot(event eventstore.Event, snapshot interface{}) error {
	data, err := json.Marshal(snapshot)
	if err != nil {
		return err
	}

	docID := fmt.Sprintf("%s-%d", event.ID, event.Version)
	wrapper := new(docWrapper)
	if err = s.db.Get(docID, wrapper, nil); err != nil {
		return err
	}
	att := new(couchdb.Attachment)
	att.Body = bytes.NewBuffer(data)
	att.Type = "application/json"
	att.Name = "snapshot"
	_, err = s.db.PutAttachment(wrapper.ID, att, wrapper.Rev)
	return err
}

func paginateView(
	db *couchdb.DB,
	input *paginateViewInput,
	callback paginateViewCallback,
) (err error) {
	for {
		page := new(viewPage)
		if err := db.View(input.DesignDoc, input.View, page, input.Options); err != nil {
			return err
		}

		lastPage := len(page.Rows) < input.Options["limit"].(int)
		if getNext := callback(page, lastPage); !getNext || lastPage {
			return nil
		}

		input.Options["start_key"] = page.Rows[len(page.Rows)-1].Key
		input.Options["skip"] = 1
	}
}

func resolveSnapshot(db *couchdb.DB, wrapper *docWrapper) (bool, error) {
	if wrapper.Attachments == nil {
		return false, nil
	}

	if _, ok := wrapper.Attachments["snapshot"]; !ok {
		return false, nil
	}

	log.Println("Attempting to load from snapshot")

	attachment, err := db.Attachment(
		wrapper.ID,
		"snapshot",
		wrapper.Rev,
	)
	if err != nil {
		return false, err
	}

	data, err := ioutil.ReadAll(attachment.Body)
	if err != nil {
		return false, err
	}

	log.Println("Snapdata: ", string(data))

	raw := json.RawMessage(data)
	wrapper.Event.Snapshot = &raw

	return true, nil
}
