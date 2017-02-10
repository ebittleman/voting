package main

import (
	"log"
	"net/http"
	"os"

	"github.com/ebittleman/voting/bus"
	"github.com/ebittleman/voting/bus/ironmq"
	"github.com/ebittleman/voting/eventmanager"
	votingCouchdb "github.com/ebittleman/voting/eventstore/couchdb"
	"github.com/ebittleman/voting/voting"
	"github.com/ebittleman/voting/voting/commands"
	"github.com/ebittleman/voting/voting/model"
	couchdb "github.com/fjl/go-couchdb"
	uuid "github.com/satori/go.uuid"
)

func main() {
	if code := run(); code != 0 {
		os.Exit(code)
	}
}

func run() int {
	// // get a directory to put our json files in
	// conn, err := jsondb.Open("./.data")
	// if err != nil {
	// 	log.Println("Fatal: ", err)
	// 	return 1
	// }
	// defer conn.Close()
	//
	// // creates an event store that will write to event.json when it closes
	// eventStore, err := json.New(conn)
	// if err != nil {
	// 	log.Println("Fatal: ", err)
	// 	return 1
	// }

	url := os.Getenv("COUCHDB_URL")
	client, err := couchdb.NewClient(url, http.DefaultTransport)
	if err != nil {
		log.Println("Fatal: ", err)
		return 1
	}

	eventStore, err := votingCouchdb.New(client)
	if err != nil {
		log.Println("Fatal: ", err)
		return 1
	}

	// component that routes events in the local process
	eventManager := eventmanager.New()
	defer eventManager.Close()

	// Forward all events to a message queue
	mq := ironmq.New("dev-queue")
	forwarder := bus.NewFowarder(mq, voting.EventTypes)
	defer forwarder.Close()
	forwarder.Subscribe(eventManager)

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
