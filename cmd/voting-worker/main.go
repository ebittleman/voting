package main

import (
	"bytes"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"github.com/ebittleman/voting/eventstore"

	"github.com/ebittleman/voting/bus"
	"github.com/ebittleman/voting/bus/ironmq"
	jsondb "github.com/ebittleman/voting/database/json"
	"github.com/ebittleman/voting/dispatcher"
	"github.com/ebittleman/voting/eventmanager"
	"github.com/ebittleman/voting/eventstore/json"
	"github.com/ebittleman/voting/views"
	"github.com/ebittleman/voting/voting"
	"github.com/ebittleman/voting/voting/handlers"
	votingViews "github.com/ebittleman/voting/voting/views"
)

func main() {
	if code := run(); code != 0 {
		os.Exit(code)
	}
}

func run() int {
	// get a directory to put our json files in
	conn, err := jsondb.Open("./.data")
	if err != nil {
		log.Println("Fatal: ", err)
		return 1
	}
	defer conn.Close()
	setEventsReadOnly(conn)

	// creates an event store that will write to event.json when it closes
	eventStore, err := json.New(conn)
	if err != nil {
		log.Println("Fatal: ", err)
		return 1
	}

	// creates a simple json table for store view data
	viewsTable, err := views.NewTable(conn)
	if err != nil {
		log.Println("Fatal: ", err)
		return 1
	}

	// processes the current event store and builds a view of all "Open" polls
	openPollsView, err := votingViews.NewOpenPolls(eventStore)
	if err != nil {
		log.Println("Fatal: ", err)
		return 1
	}
	defer openPollsView.Close()

	// component that routes events in the local process
	eventManager := eventmanager.New()
	defer eventManager.Close()

	// listens for PollOpened and PollClosed events, triggers the openPollsView
	// to rebuild, then saves it to the viewsTable
	openPollsHandler := handlers.NewOpenPolls(openPollsView, viewsTable, eventManager)
	defer openPollsHandler.Close()

	// get a message queue to retrieve messages from.
	mq := ironmq.New("dev-queue")

	// update the view table with the latest data from the view.
	refreshViewTable(openPollsHandler)

	d := dispatcher.NewBusDispatcher(
		mq,
		eventManager,
		eventTypeFilter{
			AllowedEventTypes: []string{
				"PollOpened",
				"PollClosed",
			},
		},
		&refreshFilter{
			EventStore: eventStore,
		},
	)

	// write the views file to disk every couple of seconds
	go func() {
		for {
			select {
			case <-time.After(2 * time.Second):
				if flushErr := conn.Flush(); flushErr != nil {
					log.Println("Error: Couldn't write db to disk: ", flushErr)
					d.Close()
					return
				}
			}
			log.Println("Info: wrote db to disk.")
		}
	}()

	if err = <-d.Run(); err != nil {
		log.Println("Fatal: ", err)
		return 1
	}

	return 0
}

type refreshFilter struct {
	EventStore eventstore.EventStore
}

func (r *refreshFilter) Filter(msg bus.Message) error {
	return r.EventStore.Refresh()
}

type eventTypeFilter struct {
	AllowedEventTypes []string
}

func (e eventTypeFilter) Filter(msg bus.Message) error {
	for _, eventType := range e.AllowedEventTypes {
		if eventType == msg.Event().Type {
			return nil
		}
	}

	return dispatcher.ErrNack
}

func refreshViewTable(handler handlers.PollOpenedHandler) {
	handler.PollOpenedHandler(voting.PollOpened{})
}

// setEventsReadOnly dirty little function to skip writing the events file.
func setEventsReadOnly(c *jsondb.Connection) {
	fileCreator := c.GetFileCreator()
	c.SetFileCreator(func(f string) (io.Writer, error) {
		if !strings.Contains(f, "events") {
			return fileCreator(f)
		}

		return &bytes.Buffer{}, nil
	})
}
