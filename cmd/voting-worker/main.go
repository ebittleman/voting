package main

import (
	"log"

	"github.com/ebittleman/voting/voting/app"
)

func main() {
	votingWorker := app.NewVotingWorker(
		app.VotingWorkerConfig{
			IronQueueName: "dev-queue",
			JSONDir:       "./.data",
		},
	)
	defer votingWorker.Close()

	dispatcher, err := votingWorker.Dispatcher()
	if err != nil {
		log.Fatalln("Fatal: ", err)
		return
	}

	subscribers, err := votingWorker.Subscribers()
	if err != nil {
		log.Fatalln("Fatal: ", err)
		return
	}

	if err = dispatcher.Run(subscribers...); err != nil {
		log.Fatalln("Fatal: ", err)
		return
	}
}
