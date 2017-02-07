package commands

import (
	"errors"

	"github.com/ebittleman/voting/eventmanager"
	"github.com/ebittleman/voting/eventstore"
	"github.com/ebittleman/voting/voting/model"
)

var (
	// ErrPollAlreadyExists returned if a newly generated poll id already exists
	ErrPollAlreadyExists = errors.New("Error creating poll, id already exists")
	// ErrPollNotFound returned if a poll was expected to exist but could not
	// be found
	ErrPollNotFound = errors.New("Poll not found")
)

// CreatePoll creates a new poll.
type CreatePoll struct {
	eventStore   eventstore.EventStore
	eventManager eventmanager.EventManager
	ID           string
	Issues       []model.Issue
}

// NewCreatePoll initializes a new CreatePoll command for execution.
func NewCreatePoll(
	eventStore eventstore.EventStore,
	eventManager eventmanager.EventManager,
	id string,
	issues []model.Issue,
) Command {
	return CreatePoll{
		eventStore:   eventStore,
		eventManager: eventManager,
		ID:           id,
		Issues:       issues,
	}
}

// Run executes CreatePoll command
func (c CreatePoll) Run() error {

	if events, err := c.eventStore.Query(c.ID); err != nil {
		return err
	} else if len(events) > 0 {
		return ErrPollAlreadyExists
	}

	poll := model.LoadPoll(c.ID, nil)
	for _, issue := range c.Issues {
		poll.AppendIssue(issue)
	}

	events := poll.Flush()
	if err := poll.Commit(c.eventStore, events); err != nil {
		return err
	}

	for _, event := range events {
		c.eventManager.Publish(event)
	}

	return nil
}

// OpenPoll opens a poll for accepting ballots
type OpenPoll struct {
	eventStore   eventstore.EventStore
	eventManager eventmanager.EventManager
	ID           string
}

// NewOpenPoll initializes a new OpenPoll command for execution
func NewOpenPoll(
	eventStore eventstore.EventStore,
	eventManager eventmanager.EventManager,
	id string,
) Command {
	return OpenPoll{
		eventStore:   eventStore,
		eventManager: eventManager,
		ID:           id,
	}
}

// Run executes OpenPoll command
func (o OpenPoll) Run() error {
	var (
		events eventstore.Events
		err    error
	)
	if events, err = o.eventStore.Query(o.ID); err != nil {
		return err
	} else if len(events) < 1 {
		return ErrPollNotFound
	}

	poll := model.LoadPoll(o.ID, events)
	if err = poll.OpenPolls(); err != nil {
		return err
	}

	events = poll.Flush()
	if err = poll.Commit(o.eventStore, events); err != nil {
		return err
	}

	for _, event := range events {
		o.eventManager.Publish(event)
	}

	return nil
}

// ClosePoll closes a poll so that no more ballots are accepted
type ClosePoll struct{}

// Run executes ClosePoll command
func (c ClosePoll) Run() error {
	return ErrNotImplemented
}

// CastBallot adds a vote to the poll
type CastBallot struct{}

// Run executes CastBallot command
func (c CastBallot) Run() error {
	return ErrNotImplemented
}
