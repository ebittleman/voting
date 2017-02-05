package views

import (
	"testing"
	"time"

	jsondb "github.com/ebittleman/voting/database/json"
	"github.com/ebittleman/voting/eventmanager"
	"github.com/ebittleman/voting/eventstore"
	"github.com/ebittleman/voting/eventstore/json"
)

func TestOpenPoll(t *testing.T) {
	conn, _ := jsondb.Open(".")
	defer conn.Close()

	eventStore, err := json.New(conn)
	if err != nil {
		t.Fatal(err)
	}

	eventManager := eventmanager.New()

	openPolls, err := NewOpenPolls(eventStore, eventManager)
	if err != nil {
		t.Fatal(err)
	}

	for _, id := range openPolls.List() {
		t.Log(id)
	}

	events := eventstore.Events{
		eventstore.Event{
			ID:        "poll4",
			Version:   3,
			Type:      "PollClosed",
			Timestamp: time.Now().UTC().Unix(),
		},
	}

	for _, event := range events {
		if err := eventStore.Put("poll4", event.Version-1, event); err != nil {
			t.Fatal(err)
		}
	}

	for _, event := range events {
		eventManager.Publish(event)
	}

	time.Sleep(time.Millisecond * 10)

	for _, id := range openPolls.List() {
		t.Log(id)
	}
}
