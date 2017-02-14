package model

import (
	"encoding/json"
	"errors"
	"log"
	"sort"

	"github.com/ebittleman/voting/eventstore"
	"github.com/ebittleman/voting/voting"
)

var (
	// ErrIssueNotOnPoll returned when a Ballot is cast with a Selection to an
	// Issue not on the poll.
	ErrIssueNotOnPoll = errors.New("Issue on submitted ballot not on poll")
	// ErrPollClosed is returned what a Ballot is cast when the poll is not open.
	ErrPollClosed = errors.New("Must wait for polls to be open")
	// ErrPollAlreadyOpen is returned when trying to open a poll that is already
	// open
	ErrPollAlreadyOpen = errors.New("Poll already open")
	// ErrPollAlreadyClose returned when attempting to close a closed poll
	ErrPollAlreadyClose = errors.New("Poll already closed")
)

// Issue describes a question being asked in a poll.
type Issue struct {
	Topic      string   `json:"topic"`
	Choices    []string `json:"choices"`
	CanWriteIn bool     `json:"can_write_in"`
}

// Selection describes the choice or write in of an issue.
type Selection struct {
	Issue   Issue  `json:"issue"`
	WroteIn bool   `json:"wrote_in"`
	Choice  int    `json:"choice"`
	WriteIn string `json:"write_in"`
	Comment string `json:"comment"`
}

// Ballot list of sections for a poll.
type Ballot []Selection

// Poll aggregate root in the voting system
type Poll struct {
	IsOpen bool `json:"is_open"`

	Issues  []Issue  `json:"issues"`
	Ballots []Ballot `json:"ballots"`

	AggregateRoot
}

// AppendIssue adds an issue to a poll. Can only be done if the poll is not
// open.
func (p *Poll) AppendIssue(issue Issue) {
	if p.IsOpen {
		return
	}

	for _, existingIssue := range p.Issues {
		if existingIssue.Topic == issue.Topic {
			return
		}
	}

	p.Issues = append(p.Issues, issue)

	p.Emit(issueAppended(issue.Topic, issue.Choices, issue.CanWriteIn))
}

// OpenPolls opens a poll allowing for ballots to be placed.
func (p *Poll) OpenPolls() error {
	if p.IsOpen {
		return ErrPollAlreadyOpen
	}

	p.IsOpen = true
	p.Emit(pollOpenedEvent())

	return nil
}

// ClosePolls closes a poll stopping ballots from being placed.
func (p *Poll) ClosePolls() error {
	if !p.IsOpen {
		return ErrPollAlreadyClose
	}

	p.IsOpen = false
	p.Emit(pollClosedEvent())
	return nil
}

// CastBallot adds a list of votes. Can only be run when the poll is open. And
// Only selections to issues on in poll are allowed.
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

	p.Emit(ballotCast(ballot))

	return nil
}

// Snapshot creates a snapshot of the polls current state
func (p Poll) Snapshot() (interface{}, error) {
	return p, nil
}

// LoadPoll loads a poll by id from a list of events.
func LoadPoll(id string, events eventstore.Events) Poll {
	var poll Poll

	if len(events) < 1 {
		poll.ID = id
		poll.Emit(pollCreatedEvent(id))
		return poll
	}

	sort.Sort(events)
	for _, event := range events {

		if event.Snapshot != nil {
			log.Println("Event Has Snapshot, Num Events: ", len(events))
			if err := json.Unmarshal(*event.Snapshot, &poll); err == nil {
				continue
			} else {
				log.Println("Warn: Failed to Load From Snapshot")
			}
		}

		switch event.Type {
		case "PollCreated":
			data := new(voting.PollCreated)
			json.Unmarshal(*event.Data, data)
			poll.ID = data.ID
		case "PollOpened":
			poll.IsOpen = true
		case "PollClosed":
			poll.IsOpen = false
		case "IssueAppended":
			data := new(voting.IssueAppended)
			if err := json.Unmarshal(*event.Data, data); err != nil {
				log.Println("Error: Replaying IssueAppended: ", err)
				continue
			}
			issue := new(Issue)
			issue.Topic = data.Topic
			issue.Choices = data.Choices
			issue.CanWriteIn = data.CanWriteIn
			poll.Issues = append(poll.Issues, *issue)
		case "BallotCast":
			data := make(voting.BallotCast, 0)
			if err := json.Unmarshal(*event.Data, &data); err != nil || len(data) < 1 {
				log.Println("Error: Replaying BallotCast: ", err)
				continue
			}

			ballot := make(Ballot, 0)
			for _, eventData := range data {
				for _, issue := range poll.Issues {
					if issue.Topic == eventData.IssueTopic {
						selection := new(Selection)
						selection.Issue = issue
						selection.WroteIn = eventData.WroteIn
						selection.Choice = eventData.Choice
						selection.WriteIn = eventData.WriteIn
						selection.Comment = eventData.Comment
						ballot = append(ballot, *selection)
					}
				}
			}
			poll.Ballots = append(poll.Ballots, ballot)
		}
		poll.Version = event.Version
	}

	return poll
}

func issueAppended(
	topic string,
	choices []string,
	canWriteIn bool,
) eventstore.Event {
	eventData := voting.IssueAppended{
		Topic:      topic,
		Choices:    choices,
		CanWriteIn: canWriteIn,
	}

	bytes, _ := json.Marshal(eventData)
	data := json.RawMessage(bytes)
	return eventstore.Event{
		Type: "IssueAppended",
		Data: &data,
	}
}

func ballotCast(
	ballot Ballot,
) eventstore.Event {
	var ballotEvent voting.BallotCast
	for _, selection := range ballot {
		wroteIn := len(selection.WriteIn) > 0
		eventData := voting.BallotSelection{
			IssueTopic: selection.Issue.Topic,
			WroteIn:    wroteIn,
			Choice:     selection.Choice,
			WriteIn:    selection.WriteIn,
			Comment:    selection.Comment,
		}
		ballotEvent = append(ballotEvent, eventData)
	}
	bytes, _ := json.Marshal(ballotEvent)
	data := json.RawMessage(bytes)
	return eventstore.Event{
		Type: "BallotCast",
		Data: &data,
	}
}

func pollCreatedEvent(id string) eventstore.Event {
	data := json.RawMessage(`{"id": "` + id + `"}`)
	return eventstore.Event{
		Type: "PollCreated",
		Data: &data,
	}
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
