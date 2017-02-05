package json

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"
	"testing"
	"time"

	jsondb "github.com/ebittleman/voting/database/json"
	"github.com/ebittleman/voting/eventstore"
)

func TestNewStore(t *testing.T) {
	conn, _ := jsondb.Open(".")
	conn.SetFileCreator(func(f string) (io.Writer, error) {
		return bytes.NewBuffer(nil), nil
	})

	conn.SetFileProvider(func(f string) (io.Reader, error) {
		return strings.NewReader(testData), nil
	})

	store, err := New(conn)
	if err != nil {
		t.Fatal(err)
	}

	events, err := store.Query("id")
	if err != nil {
		t.Fatal(err)
	}

	var event eventstore.Event
	for _, event = range events {
		t.Log(event)
	}

	msg := json.RawMessage(`{"foo": "bar"}`)
	if err := store.Put("id", event.Version, eventstore.Event{
		ID:        "id",
		Version:   event.Version + 1,
		Type:      "NewItem",
		Timestamp: time.Now().UTC().Unix(),
		Data:      &msg,
	}); err != nil {
		t.Fatal(err)
	}

	conn.Close()
}

const testData = `{"id":"id","version":1,"type":"NewItem","timestamp":1486332029,"data":{"foo":"bar"}}
{"id":"id","version":2,"type":"NewItem","timestamp":1486332324,"data":{"foo":"bar"}}
{"id":"id","version":3,"type":"NewItem","timestamp":1486332354,"data":{"foo":"bar"}}
{"id":"id","version":4,"type":"NewItem","timestamp":1486332418,"data":{"foo":"bar"}}
`
