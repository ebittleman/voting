package handlers

import (
	"encoding/json"
	"io"
	"log"

	"github.com/ebittleman/voting/eventmanager"
	"github.com/ebittleman/voting/eventstore"
	"github.com/ebittleman/voting/voting"
)

var eventTypes = []string{
	"PollCreated",
	"PollOpened",
	"PollClosed",
	"IssueAppended",
	"BallotCast",
}

// Subscribe registers all default handlers to an event manager
func Subscribe(handler interface{}, em eventmanager.EventManager) EventWrapper {
	wrapper := new(eventWrapper)
	wrapper.eventManager = em
	wrapper.handler = handler

	for _, eventType := range eventTypes {
		wrapper.subs = append(
			wrapper.subs,
			em.Subscribe(eventType, wrapper.EventHandler),
		)
	}

	return wrapper
}

// PollCreatedHandler handlers PollCreated events.
type PollCreatedHandler interface {
	PollCreatedHandler(event voting.PollCreated) error
}

// PollOpenedHandler handlers PollOpened events.
type PollOpenedHandler interface {
	PollOpenedHandler(event voting.PollOpened) error
}

// PollClosedHandler handlers PollClosed events.
type PollClosedHandler interface {
	PollClosedHandler(event voting.PollClosed) error
}

// IssueAppendedHandler handlers IssueAppended events.
type IssueAppendedHandler interface {
	IssueAppendedHandler(event voting.IssueAppended) error
}

// BallotCastHandler handlers BallotCast events.
type BallotCastHandler interface {
	BallotCastHandler(event voting.BallotCast) error
}

// EventWrapper routes all model events to an attached handler.
type EventWrapper interface {
	EventHandler(event eventstore.Event) error
	io.Closer
}

type eventWrapper struct {
	handler      interface{}
	eventManager eventmanager.EventManager
	subs         []eventmanager.Subscription
}

// EventHandler routes events from events published by an event manager
func (p *eventWrapper) EventHandler(event eventstore.Event) error {
	switch event.Type {
	case "PollCreated":
		return p.PollCreatedHandler(voting.PollCreated{
			ID: event.ID,
		})
	case "PollOpened":
		return p.PollOpenedHandler(voting.PollOpened{
			ID: event.ID,
		})
	case "PollClosed":
		return p.PollClosedHandler(voting.PollClosed{
			ID: event.ID,
		})
	case "IssueAppended":
		issueAppended := new(voting.IssueAppended)
		if err := json.Unmarshal(event.Data, issueAppended); err != nil {
			return err
		}

		return p.IssueAppendedHandler(*issueAppended)
	case "BallotCast":
		ballotCast := make(voting.BallotCast, 0)
		if err := json.Unmarshal(event.Data, &ballotCast); err != nil {
			return err
		}

		return p.BallotCastHandler(ballotCast)
	}

	return eventmanager.ErrUnhandledEventType
}

func (p *eventWrapper) Close() error {
	for _, sub := range p.subs {
		if err := p.eventManager.Unsubscribe(sub); err != nil {
			log.Println("Warn: Unsubscribing Handler: ", err)
		}
	}

	return nil
}

// PollCreatedHandler handlers PollCreated events.
func (p *eventWrapper) PollCreatedHandler(event voting.PollCreated) error {
	log.Println("Info: Handle Poll Created")
	if handler, ok := p.handler.(PollCreatedHandler); ok {
		return handler.PollCreatedHandler(event)
	}
	return nil
}

// PollOpenedHandler handles PollOpened events
func (p *eventWrapper) PollOpenedHandler(event voting.PollOpened) error {
	log.Println("Info: Handle Poll Opened")
	if handler, ok := p.handler.(PollOpenedHandler); ok {
		return handler.PollOpenedHandler(event)
	}
	return nil
}

// PollClosedHandler handles PollClosed events
func (p *eventWrapper) PollClosedHandler(event voting.PollClosed) error {
	log.Println("Info: Handle Poll Closed")
	if handler, ok := p.handler.(PollClosedHandler); ok {
		return handler.PollClosedHandler(event)
	}
	return nil
}

// IssueAppendedHandler handles PollClosed events
func (p *eventWrapper) IssueAppendedHandler(event voting.IssueAppended) error {
	log.Println("Info: Handle Issue Appended")
	if handler, ok := p.handler.(IssueAppendedHandler); ok {
		return handler.IssueAppendedHandler(event)
	}
	return nil
}

// BallotCastHandler handles PollClosed events
func (p *eventWrapper) BallotCastHandler(event voting.BallotCast) error {
	log.Println("Info: Handle Ballot Cast")
	if handler, ok := p.handler.(BallotCastHandler); ok {
		return handler.BallotCastHandler(event)
	}
	return nil
}
