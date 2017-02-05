package json

import (
	"testing"
	"time"

	"github.com/ebittleman/voting/database/json"
	"github.com/ebittleman/voting/eventstore"
)

func TestNewStore(t *testing.T) {
	conn, _ := json.Open(".")
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

	if err := store.Put("id", event.Version, eventstore.Event{
		ID:        "id",
		Version:   event.Version + 1,
		Type:      "NewItem",
		Timestamp: time.Now().UTC().Unix(),
		Data:      []byte(`{"foo": "bar"}`),
	}); err != nil {
		t.Fatal(err)
	}

	conn.Close()
}
