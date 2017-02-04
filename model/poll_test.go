package model

import (
	"testing"

	jsondb "code.bittles.net/voting/database/json"
	"code.bittles.net/voting/eventstore/json"
)

func TestAddIssue(t *testing.T) {
	var poll Poll

	issue := Issue{
		Topic: "What do you want for dinner?",
	}

	poll.AddIssue(issue)

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

	poll.AddIssue(issue)
	poll.AddIssue(issue)

	if len(poll.Issues) != 1 {
		t.Fatalf("Expected: 2 item, Got: %d item(s)", len(poll.Issues))
	}

	if poll.Issues[0].Topic != issue.Topic {
		t.Fatalf("Expected: `%s`, Got: `%s`", issue.Topic, poll.Issues[0].Topic)
	}
}

func TestCastBallot(t *testing.T) {
	id := "poll2"
	conn, _ := jsondb.Open(".")
	defer conn.Close()
	store := json.New(conn)
	events, _ := store.Query(id)
	poll := LoadPoll(id, events)

	var ballot Ballot

	poll.AddIssue(Issue{
		Topic:   "What's for lunch?",
		Choices: []string{"Soup", "Sandwich"},
	})

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

	newEvents := poll.events
	if len(newEvents) != 2 {
		t.Fatalf("Expected: 2 events, Got: %d event(s)", len(newEvents))
	}

	if newEvents[0].Type != "PollOpened" {
		t.Fatalf("Expected: `%s`, Got: `%s`", "PollOpened", newEvents[0].Type)
	}

	if newEvents[1].Type != "PollClosed" {
		t.Fatalf("Expected: `%s`, Got: `%s`", "PollClosed", newEvents[1].Type)
	}

	if err := poll.Commit(store); err != nil {
		t.Fatal(err)
	}
}
