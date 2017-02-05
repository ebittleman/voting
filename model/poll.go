package model

import (
	"errors"

	"github.com/ebittleman/voting/eventstore"
)

var (
	ErrIssueNotOnPoll = errors.New("Issue on submitted ballot not on poll.")
	ErrPollClosed     = errors.New("Must wait for polls to be open.")
)

// Issue ...
type Issue struct {
	Topic      string
	Choices    []string
	CanWriteIn bool
}

// Selection ...
type Selection struct {
	Issue   Issue
	WroteIn bool
	Choice  int
	WriteIn []byte
	Comment []byte
}

// Ballot ..
type Ballot []Selection

// Poll aggregate root in the voting system
type Poll struct {
	IsOpen bool

	Issues  []Issue
	Ballots []Ballot

	AggregateRoot
}

// AddIssue ...
func (p *Poll) AddIssue(issue Issue) {
	if p.IsOpen {
		return
	}

	for _, existingIssue := range p.Issues {
		if existingIssue.Topic == issue.Topic {
			return
		}
	}

	p.Issues = append(p.Issues, issue)

	// p.Emit()
}

// OpenPolls ...
func (p *Poll) OpenPolls() {
	if p.IsOpen {
		return
	}

	p.IsOpen = true

	p.Emit(pollOpenedEvent())
}

// ClosePolls ...
func (p *Poll) ClosePolls() {
	if !p.IsOpen {
		return
	}

	p.IsOpen = true
	p.Emit(pollClosedEvent())
}

// CastBallot ..
func (p *Poll) CastBallot(ballot Ballot) error {
	if !p.IsOpen {
		return ErrPollClosed
	}

	for _, selection := range ballot {
		found := false
		for _, issue := range p.Issues {
			if selection.Issue.Topic == issue.Topic {
				found = true
				break
			}
		}
		if !found {
			return ErrIssueNotOnPoll
		}
	}

	p.Ballots = append(p.Ballots, ballot)

	return nil
}

func LoadPoll(id string, events eventstore.Events) Poll {
	var poll Poll
	poll.ID = id
	for _, event := range events {
		switch event.Type {
		case "PollOpened":
			poll.IsOpen = true
		case "PollClosed":
			poll.IsOpen = false
		}
		poll.version = event.Version
	}

	return poll
}

func pollOpenedEvent() eventstore.Event {
	return eventstore.Event{
		Type: "PollOpened",
	}
}

func pollClosedEvent() eventstore.Event {
	return eventstore.Event{
		Type: "PollClosed",
	}
}
