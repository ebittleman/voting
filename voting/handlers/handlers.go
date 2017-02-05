package handlers

import (
	"github.com/ebittleman/voting/eventmanager"
	"github.com/ebittleman/voting/eventstore"
	"github.com/ebittleman/voting/voting"
)

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
	return nil
}

func PollOpenedHandler(event voting.PollOpened) error {
	return nil
}

func PollClosedHandler(event voting.PollClosed) error {
	return nil
}
