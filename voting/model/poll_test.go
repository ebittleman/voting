package model

import (
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

	// i := 0
	// newEvents := poll.events
	// if len(newEvents) != (expectedLen + offset) {
	// 	t.Fatalf(
	// 		"Expected: %d events, Got: %d event(s)",
	// 		(expectedLen + offset),
	// 		len(newEvents),
	// 	)
	// }
	//
	// if !hasCreated {
	// 	if newEvents[i].Type != "PollCreated" {
	// 		t.Fatalf("Expected: `%s`, Got: `%s`", "PollCreated", newEvents[i].Type)
	// 	}
	// 	i++
	// }
	//
	// if newEvents[i].Type != "PollOpened" {
	// 	t.Fatalf("Expected: `%s`, Got: `%s`", "PollOpened", newEvents[i].Type)
	// }
	// i++
	//
	// if newEvents[i].Type != "PollClosed" {
	// 	t.Fatalf("Expected: `%s`, Got: `%s`", "PollClosed", newEvents[i].Type)
	// }

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

// func TestBuild(t *testing.T) {
// 	conn, _ := jsondb.Open(".")
// 	defer conn.Close()
//
// 	id := "poll3"
// 	store, err := json.New(conn)
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	events, _ := store.Query(id)
//
// 	poll := LoadPoll(id, events)
// 	// poll.OpenPolls()
// 	poll.ClosePolls()
//
// 	events = poll.Flush()
// 	if err := poll.Commit(store, events); err != nil {
// 		t.Fatal(err)
// 	}
//
// 	em := eventmanager.New()
// 	handlers.Subscribe(nil, em)
//
// 	for _, event := range events {
// 		em.Publish(event)
// 	}
// 	em.Close()
// }
