package views

import (
	"bytes"
	"io"
	"strings"
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

	conn.SetFileCreator(func(f string) (io.Writer, error) {
		return bytes.NewBuffer(nil), nil
	})

	conn.SetFileProvider(func(f string) (io.Reader, error) {
		return strings.NewReader(testData), nil
	})

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

	id := "poll4"

	events := eventstore.Events{
		eventstore.Event{
			ID:        id,
			Version:   4,
			Type:      "PollClosed",
			Timestamp: time.Now().UTC().Unix(),
		},
	}

	for _, event := range events {
		if err := eventStore.Put(id, event.Version-1, event); err != nil {
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

const testData = `{"id":"poll2","version":1,"type":"PollCreated","timestamp":1486332029,"data":{"id":"poll2"}}
{"id":"poll2","version":2,"type":"IssueAppended","timestamp":1486332029,"data":{"topic":"What's for lunch?","choices":["Soup","Sandwich"],"can_write_in":false}}
{"id":"poll2","version":3,"type":"PollOpened","timestamp":1486332029,"data":null}
{"id":"poll2","version":4,"type":"BallotCast","timestamp":1486332029,"data":[{"issue_topic":"What's for lunch?","wrote_in":false,"choice":0,"write_in":"","comment":""}]}
{"id":"poll2","version":5,"type":"PollClosed","timestamp":1486332029,"data":null}
{"id":"poll2","version":6,"type":"PollOpened","timestamp":1486332249,"data":null}
{"id":"poll2","version":7,"type":"BallotCast","timestamp":1486332249,"data":[{"issue_topic":"What's for lunch?","wrote_in":false,"choice":0,"write_in":"","comment":""}]}
{"id":"poll2","version":8,"type":"PollClosed","timestamp":1486332249,"data":null}
{"id":"poll2","version":9,"type":"PollOpened","timestamp":1486332249,"data":null}
{"id":"poll2","version":10,"type":"BallotCast","timestamp":1486332249,"data":[{"issue_topic":"What's for lunch?","wrote_in":false,"choice":0,"write_in":"","comment":""}]}
{"id":"poll2","version":11,"type":"PollClosed","timestamp":1486332249,"data":null}
{"id":"poll2","version":12,"type":"PollOpened","timestamp":1486332255,"data":null}
{"id":"poll2","version":13,"type":"BallotCast","timestamp":1486332255,"data":[{"issue_topic":"What's for lunch?","wrote_in":false,"choice":0,"write_in":"","comment":""}]}
{"id":"poll2","version":14,"type":"PollClosed","timestamp":1486332255,"data":null}
{"id":"poll2","version":15,"type":"PollOpened","timestamp":1486332255,"data":null}
{"id":"poll2","version":16,"type":"BallotCast","timestamp":1486332255,"data":[{"issue_topic":"What's for lunch?","wrote_in":false,"choice":0,"write_in":"","comment":""}]}
{"id":"poll2","version":17,"type":"PollClosed","timestamp":1486332255,"data":null}
{"id":"poll2","version":18,"type":"PollOpened","timestamp":1486332324,"data":null}
{"id":"poll2","version":19,"type":"BallotCast","timestamp":1486332324,"data":[{"issue_topic":"What's for lunch?","wrote_in":false,"choice":0,"write_in":"","comment":""}]}
{"id":"poll2","version":20,"type":"PollClosed","timestamp":1486332324,"data":null}
{"id":"poll4","version":3,"type":"PollOpened","timestamp":1486332455,"data":null}
{"id":"poll5","version":3,"type":"PollOpened","timestamp":1486332483,"data":null}
{"id":"poll6","version":3,"type":"PollOpened","timestamp":1486332491,"data":null}
`
