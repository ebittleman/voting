package app

import (
	"bytes"
	"io"
	"log"
	"strings"
	"time"

	"github.com/ebittleman/voting/bus"
	"github.com/ebittleman/voting/bus/ironmq"
	jsondb "github.com/ebittleman/voting/database/json"
	"github.com/ebittleman/voting/dispatcher"
	"github.com/ebittleman/voting/dispatcher/filters"
	"github.com/ebittleman/voting/eventmanager"
	"github.com/ebittleman/voting/eventstore"
	"github.com/ebittleman/voting/eventstore/json"
	"github.com/ebittleman/voting/views"
	jsonViews "github.com/ebittleman/voting/views/json"
	"github.com/ebittleman/voting/voting"
	"github.com/ebittleman/voting/voting/handlers"
	votingViews "github.com/ebittleman/voting/voting/views"
)

// VotingWorkerConfig of a voting-working application
type VotingWorkerConfig struct {
	IronQueueName string
	JSONDir       string
}

type votingWorker struct {
	jsonDir       string
	ironQueueName string

	conn             *jsondb.Connection
	dispatcher       dispatcher.Runnable
	eventManager     eventmanager.EventManager
	eventStore       eventstore.EventStore
	filters          []dispatcher.Filter
	mq               bus.MessageQueue
	openPollsView    *votingViews.OpenPolls
	openPollsHandler eventmanager.Subscriber
	subscribers      []eventmanager.Subscriber
	viewStore        views.ViewStore

	closers []io.Closer
}

// NewVotingWorker initializes the compnents of a a voting-working application
func NewVotingWorker(config VotingWorkerConfig) Application {
	c := new(votingWorker)
	c.ironQueueName = config.IronQueueName
	c.jsonDir = config.JSONDir

	return c
}

func (c *votingWorker) Close() error {
	for _, closer := range c.closers {
		closer.Close()
	}

	return nil
}

func (c *votingWorker) Subscribers() ([]eventmanager.Subscriber, error) {
	if c.subscribers != nil {
		return c.subscribers, nil
	}

	openPollsHandler, err := c.OpenPollsHandler()
	if err != nil {
		return nil, err
	}

	c.subscribers = []eventmanager.Subscriber{
		openPollsHandler,
	}

	return c.subscribers, nil
}

func (c *votingWorker) Filters() ([]dispatcher.Filter, error) {
	if c.filters != nil {
		return c.filters, nil
	}

	eventStore, err := c.EventStore()
	if err != nil {
		return nil, err
	}

	c.filters = []dispatcher.Filter{
		filters.EventTypeFilter{
			AllowedEventTypes: []string{
				"PollOpened",
				"PollClosed",
			},
		},
		&filters.RefreshFilter{
			EventStore: eventStore,
		},
	}

	return c.filters, nil
}

func (c *votingWorker) Dispatcher() (dispatcher.Runnable, error) {
	if c.dispatcher != nil {
		return c.dispatcher, nil
	}

	filters, err := c.Filters()
	if err != nil {
		return nil, err
	}

	c.dispatcher = dispatcher.NewBusDispatcher(
		c.MQ(),
		c.EventManager(),
		filters...,
	)

	c.closers = append(c.closers, c.dispatcher)

	return c.dispatcher, nil
}

func (c *votingWorker) Connection() (*jsondb.Connection, error) {
	if c.conn != nil {
		return c.conn, nil
	}

	conn, err := jsondb.Open(c.jsonDir)
	if err != nil {
		return nil, err
	}
	// setEventsReadOnly dirty little trick to skip writing the events file.
	fileCreator := conn.GetFileCreator()
	conn.SetFileCreator(func(f string) (io.Writer, error) {
		if !strings.Contains(f, "events") {
			return fileCreator(f)
		}

		return &bytes.Buffer{}, nil
	})

	c.closers = append(c.closers, conn)
	c.conn = conn

	// write the views file to disk every couple of seconds
	go func() {
		for {
			select {
			case <-time.After(2 * time.Second):
				if flushErr := conn.Flush(); flushErr != nil {
					log.Println("Error: Couldn't write db to disk: ", flushErr)
					return
				}
			}
			log.Println("Info: wrote db to disk.")
		}
	}()

	return c.conn, nil
}

func (c *votingWorker) EventManager() eventmanager.EventManager {
	if c.eventManager != nil {
		return c.eventManager
	}

	c.eventManager = eventmanager.New()
	c.closers = append(c.closers, c.eventManager)

	return c.eventManager
}

func (c *votingWorker) EventStore() (eventstore.EventStore, error) {
	if c.eventStore != nil {
		return c.eventStore, nil
	}

	conn, err := c.Connection()
	if err != nil {
		return nil, err
	}

	store, err := json.New(conn)
	if err != nil {
		return nil, err
	}
	c.eventStore = store

	return c.eventStore, nil
}

func (c *votingWorker) MQ() bus.MessageQueue {
	if c.mq != nil {
		return c.mq
	}

	c.mq = ironmq.New(c.ironQueueName)

	return c.mq
}

func (c *votingWorker) OpenPollsView() (*votingViews.OpenPolls, error) {
	if c.openPollsView != nil {
		return c.openPollsView, nil
	}

	eventStore, err := c.EventStore()
	if err != nil {
		return nil, err
	}

	openPollsView, err := votingViews.NewOpenPolls(eventStore)
	if err != nil {
		return nil, err
	}
	c.openPollsView = openPollsView
	c.closers = append(c.closers, c.openPollsView)

	return c.openPollsView, nil
}

func (c *votingWorker) OpenPollsHandler() (eventmanager.Subscriber, error) {
	if c.openPollsHandler != nil {
		return c.openPollsHandler, nil
	}

	openPollsView, err := c.OpenPollsView()
	if err != nil {
		return nil, err
	}

	viewStore, err := c.ViewStore()
	if err != nil {
		return nil, err
	}

	openPollsHandler := handlers.NewOpenPolls(openPollsView, viewStore)
	openPollsHandler.PollOpenedHandler(voting.PollOpened{})

	c.openPollsHandler = openPollsHandler
	c.closers = append(c.closers, c.openPollsHandler)

	return c.openPollsHandler, nil
}

func (c *votingWorker) ViewStore() (views.ViewStore, error) {
	if c.viewStore != nil {
		return c.viewStore, nil
	}

	conn, err := c.Connection()
	if err != nil {
		return nil, err
	}

	viewStore, err := jsonViews.NewStore(conn)
	if err != nil {
		return nil, err
	}
	c.viewStore = viewStore

	return viewStore, nil
}
