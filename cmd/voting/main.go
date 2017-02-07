package main

import (
	"log"
	"os"

	"github.com/ebittleman/voting/bus/ironmq"
	jsondb "github.com/ebittleman/voting/database/json"
	"github.com/ebittleman/voting/eventmanager"
	"github.com/ebittleman/voting/eventstore/json"
	"github.com/ebittleman/voting/voting/commands"
	"github.com/ebittleman/voting/voting/handlers"
	"github.com/ebittleman/voting/voting/model"
	uuid "github.com/satori/go.uuid"
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

	// creates an event store that will write to event.json when it closes
	eventStore, err := json.New(conn)
	if err != nil {
		log.Println("Fatal: ", err)
		return 1
	}

	// creates a simple json table for store view data
	// viewsTable, err := views.NewTable(conn)
	// if err != nil {
	// 	log.Println("Fatal: ", err)
	// 	return 1
	// }

	// processes the current event store and builds a view of all "Open" polls
	// openPollsView, err := votingViews.NewOpenPolls(eventStore)
	// if err != nil {
	// 	log.Println("Fatal: ", err)
	// 	return 1
	// }
	// defer openPollsView.Close()

	// component that routes events in the local process
	eventManager := eventmanager.New()
	defer eventManager.Close()

	// // listens for PollOpened and PollClosed events, triggers the openPollsView
	// // to rebuild, then saves it to the viewsTable
	// openPollsHandler := handlers.NewOpenPolls(openPollsView, viewsTable, eventManager)
	// defer openPollsHandler.Close()

	// Forward all events to a message queue
	bus := ironmq.New("dev-queue")
	forwarder := handlers.NewFowarder(bus, eventManager)
	defer forwarder.Close()

	// generate a new poll id
	id := uuid.NewV4().String()

	// initialize a new CreatePoll command
	createPoll := commands.NewCreatePoll(
		eventStore,
		eventManager,
		id,
		[]model.Issue{
			model.Issue{
				Topic: "What's for dinner?",
				Choices: []string{
					"Chicken",
					"Beef",
					"Veggies",
				},
			},
		},
	)

	// create the poll
	err = createPoll.Run()
	if err != nil {
		log.Println("Fatal: ", err)
		return 1
	}

	// initialize the OpenPoll command and reuse the the same id to open our
	// newly created poll
	openPoll := commands.NewOpenPoll(
		eventStore,
		eventManager,
		id,
	)

	// open the poll
	err = openPoll.Run()
	if err != nil {
		log.Println("Fatal: ", err)
		return 1
	}

	return 0
}
