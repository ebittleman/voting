package handlers

import (
	"log"

	"github.com/ebittleman/voting/eventmanager"
	"github.com/ebittleman/voting/eventstore"
	"github.com/ebittleman/voting/voting"
)

var eventTypes = []string{
	"PollCreated",
	"PollOpened",
	"PollClosed",
}

func Subscribe(em eventmanager.EventManager) {
	for _, eventType := range eventTypes {
		em.Subscribe(eventType, DefaultEventHandler)
	}
}

func DefaultEventHandler(event eventstore.Event) error {
	switch event.Type {
	case "PollCreated":
		return PollCreatedHandler(voting.PollCreated{
			ID: event.ID,
		})
	case "PollOpened":
		return PollOpenedHandler(voting.PollOpened{
			ID: event.ID,
		})
	case "PollClosed":
		return PollClosedHandler(voting.PollClosed{
			ID: event.ID,
		})
	}

	return eventmanager.ErrUnhandledEventType
}

func PollCreatedHandler(event voting.PollCreated) error {
	log.Println("Handle Poll Created")
	return nil
}

func PollOpenedHandler(event voting.PollOpened) error {
	log.Println("Handle Poll Opened")
	return nil
}

func PollClosedHandler(event voting.PollClosed) error {
	log.Println("Handle Poll Closed")
	return nil
}
