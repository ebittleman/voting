package model

import (
	"bytes"
	"io"
	"strings"
	"testing"

	jsondb "github.com/ebittleman/voting/database/json"
	"github.com/ebittleman/voting/eventmanager"
	"github.com/ebittleman/voting/eventstore/json"
	"github.com/ebittleman/voting/voting/handlers"
)

func TestAppendIssue(t *testing.T) {
	var poll Poll

	issue := Issue{
		Topic: "What do you want for dinner?",
	}

	poll.AppendIssue(issue)

	if len(poll.Issues) != 1 {
		t.Fatalf("Expected: 1 item, Got: %d item(s)", len(poll.Issues))
	}

	if poll.Issues[0].Topic != issue.Topic {
		t.Fatalf("Expected: `%s`, Got: `%s`", issue.Topic, poll.Issues[0].Topic)
	}
}

func TestAddDuplicateIssue(t *testing.T) {
	var poll Poll

	issue := Issue{
		Topic: "What do you want for dinner?",
	}

	poll.AppendIssue(issue)
	poll.AppendIssue(issue)

	if len(poll.Issues) != 1 {
		t.Fatalf("Expected: 2 item, Got: %d item(s)", len(poll.Issues))
	}

	if poll.Issues[0].Topic != issue.Topic {
		t.Fatalf("Expected: `%s`, Got: `%s`", issue.Topic, poll.Issues[0].Topic)
	}
}

func TestCastBallot(t *testing.T) {
	conn, _ := jsondb.Open(".")
	defer conn.Close()

	conn.SetFileCreator(func(f string) (io.Writer, error) {
		return bytes.NewBuffer(nil), nil
	})

	conn.SetFileProvider(func(f string) (io.Reader, error) {
		return strings.NewReader(testData), nil
	})

	store, err := json.New(conn)
	if err != nil {
		t.Fatal(err)
	}

	id := "poll2"
	events, _ := store.Query(id)

	// hasCreated := len(events) > 0
	// expectedLen, offset := 2, 0
	// if !hasCreated {
	// 	offset++
	// }

	poll := LoadPoll(id, events)

	poll.AppendIssue(Issue{
		Topic:   "What's for lunch?",
		Choices: []string{"Soup", "Sandwich"},
	})

	var ballot Ballot
	for _, issue := range poll.Issues {
		ballot = append(ballot, Selection{
			Issue:  issue,
			Choice: 0,
		})
	}

	poll.OpenPolls()
	if err := poll.CastBallot(ballot); err != nil {
		t.Fatal(err)
	}
	poll.ClosePolls()

	// Mess with saving and publishing the transaction
	events = poll.Flush()
	if err := poll.Commit(store, events); err != nil {
		t.Fatal(err)
	}

	em := eventmanager.New()
	handlers.Subscribe(nil, em)

	for _, event := range events {
		em.Publish(event)
	}
	em.Close()
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
{"id":"poll2","version":21,"type":"PollOpened","timestamp":1486332354,"data":null}
{"id":"poll2","version":22,"type":"BallotCast","timestamp":1486332354,"data":[{"issue_topic":"What's for lunch?","wrote_in":false,"choice":0,"write_in":"","comment":""}]}
{"id":"poll2","version":23,"type":"PollClosed","timestamp":1486332354,"data":null}
{"id":"poll2","version":24,"type":"PollOpened","timestamp":1486332418,"data":null}
{"id":"poll2","version":25,"type":"BallotCast","timestamp":1486332418,"data":[{"issue_topic":"What's for lunch?","wrote_in":false,"choice":0,"write_in":"","comment":""}]}
{"id":"poll2","version":26,"type":"PollClosed","timestamp":1486332418,"data":null}
`
