package main

import (
	"log"
	"os"
	"time"

	jsondb "github.com/ebittleman/voting/database/json"
	"github.com/ebittleman/voting/eventmanager"
	"github.com/ebittleman/voting/eventstore/json"
	"github.com/ebittleman/voting/views"
	"github.com/ebittleman/voting/voting/commands"
	"github.com/ebittleman/voting/voting/handlers"
	"github.com/ebittleman/voting/voting/model"
	votingViews "github.com/ebittleman/voting/voting/views"
	uuid "github.com/satori/go.uuid"
)

func main() {
	if code := run(); code != 0 {
		os.Exit(code)
	}
}

func run() int {
	conn, err := jsondb.Open("./.data")
	if err != nil {
		log.Println("Fatal: ", err)
		return 1
	}
	defer conn.Close()

	eventManager := eventmanager.New()
	defer eventManager.Close()

	eventStore, err := json.New(conn)
	if err != nil {
		log.Println("Fatal: ", err)
		return 1
	}

	viewsTable, err := views.NewTable(conn)
	if err != nil {
		log.Println("Fatal: ", err)
		return 1
	}

	openPollsView, err := votingViews.NewOpenPolls(eventStore)
	if err != nil {
		log.Println("Fatal: ", err)
		return 1
	}
	defer openPollsView.Close()

	openPollsHandler := handlers.NewOpenPolls(openPollsView, viewsTable, eventManager)
	defer openPollsHandler.Close()

	id := uuid.NewV4().String()
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

	err = createPoll.Run()
	if err != nil {
		log.Println("Fatal: ", err)
		return 1
	}

	openPoll := commands.NewOpenPoll(
		eventStore,
		eventManager,
		id,
	)

	err = openPoll.Run()
	if err != nil {
		log.Println("Fatal: ", err)
		return 1
	}

	time.Sleep(time.Second)

	return 0
}
