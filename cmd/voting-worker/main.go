package main

import (
	"bytes"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"github.com/ebittleman/voting/bus/ironmq"
	jsondb "github.com/ebittleman/voting/database/json"
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

	// update the view table with the latest data from the view.
	refreshViewTable(openPollsHandler)

	// write the views file to disk every couple of seconds
	go func() {
		for {
			select {
			case <-time.After(2 * time.Second):
				err := conn.Flush()
				if err != nil {
					panic(err)
				}
			}
			log.Println("Info: wrote db to disk.")
		}
	}()

	// get a message queue to retrieve messages from.
	bus := ironmq.New("dev-queue")
	for {
		// long poll on getting a message
		msg, err := bus.Receive()
		if err != nil {
			log.Println("Fatal: Receive: ", err)
			return 1
		}

		// if it isn't the type of message we want sent it back to the queue
		if msg.Event().Type != "PollOpened" && msg.Event().Type != "PollClosed" {
			if err = bus.Nack(msg); err != nil {
				log.Println("Fatal: Nack: ", err)
				return 1
			}
			continue
		}

		// ensure the event store has the latest data from disk.
		if err = eventStore.Refresh(); err != nil {
			log.Println("Fatal: Receive: ", err)
			return 1
		}

		// publish the event to the local event manager
		eventManager.Publish(msg.Event())

		// once the work has been started, ack message to the message queue.
		if err = bus.Ack(msg); err != nil {
			log.Println("Fatal: Ack ", err)
			return 1
		}
	}
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
