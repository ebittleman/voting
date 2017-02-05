package eventmanager

import (
	"sync"
	"testing"

	"github.com/ebittleman/voting/eventstore"
)

func TestPublish(t *testing.T) {
	var (
		events      eventManager
		mockHandler mockHandler
	)
	events.init()
	events.Subscribe("testEvent", mockHandler.handle)

	expectedCalls := 3
	for x := 0; x < expectedCalls; x++ {
		mockHandler.Add(1)
		events.Publish(eventstore.Event{
			Type: "testEvent",
		})
	}

	mockHandler.Wait()
	if mockHandler.called != expectedCalls {
		t.Fatalf("Expected: %d call(s), Got: %d call(s)", expectedCalls, mockHandler.called)
	}
}

func TestUnsubscribe(t *testing.T) {
	var (
		events  eventManager
		handler mockHandler
	)
	events.init()

	sub := events.Subscribe("testEvent", handler.handle)

	subs := events.subscriptions["testEvent"]
	if len(subs) != 1 {
		t.Fatalf("Expected: %d, Got: %d", 1, len(subs))
	}

	events.Unsubscribe(sub)

	subs = events.subscriptions["testEvent"]
	if len(subs) != 0 {
		t.Fatalf("Expected: %d, Got: %d", 0, len(subs))
	}
}

type mockHandler struct {
	called int
	sync.WaitGroup
}

func (m *mockHandler) handle(_ eventstore.Event) error {
	m.called++
	m.Done()
	return nil
}
