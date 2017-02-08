package main

import (
	"bytes"
	"io"
	"log"
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
	jsonViews "github.com/ebittleman/voting/views/json"
	"github.com/ebittleman/voting/voting"
	"github.com/ebittleman/voting/voting/handlers"
	votingViews "github.com/ebittleman/voting/voting/views"
)

func main() {
	var c components
	defer c.Close()

	c.ironQueueName = "dev-queue"
	c.jsonDir = "./.data"

	d, err := c.Dispatcher()
	if err != nil {
		log.Fatalln("Fatal: ", err)
		return
	}

	if err = <-d.Run(); err != nil {
		log.Fatalln("Fatal: ", err)
		return
	}
}

type components struct {
	jsonDir       string
	ironQueueName string

	conn             *jsondb.Connection
	dispatcher       dispatcher.Runnable
	eventManager     eventmanager.EventManager
	eventStore       eventstore.EventStore
	filters          []dispatcher.Filter
	mq               bus.MessageQueue
	openPollsView    *votingViews.OpenPolls
	openPollsHandler *handlers.OpenPolls
	viewStore        views.ViewStore

	closers []io.Closer
}

func (c *components) Close() error {
	for _, closer := range c.closers {
		closer.Close()
	}

	return nil
}

func (c *components) Connection() (*jsondb.Connection, error) {
	if c.conn != nil {
		return c.conn, nil
	}

	conn, err := jsondb.Open(c.jsonDir)
	if err != nil {
		return nil, err
	}
	setEventsReadOnly(conn)

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

func (c *components) Dispatcher() (dispatcher.Runnable, error) {
	if c.dispatcher != nil {
		return c.dispatcher, nil
	}

	if _, err := c.OpenPollsHandler(); err != nil {
		return nil, err
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

func (c *components) EventManager() eventmanager.EventManager {
	if c.eventManager != nil {
		return c.eventManager
	}

	c.eventManager = eventmanager.New()
	c.closers = append(c.closers, c.eventManager)

	return c.eventManager
}

func (c *components) EventStore() (eventstore.EventStore, error) {
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

func (c *components) Filters() ([]dispatcher.Filter, error) {
	if c.filters != nil {
		return c.filters, nil
	}

	eventStore, err := c.EventStore()
	if err != nil {
		return nil, err
	}

	c.filters = []dispatcher.Filter{
		eventTypeFilter{
			AllowedEventTypes: []string{
				"PollOpened",
				"PollClosed",
			},
		},
		&refreshFilter{
			EventStore: eventStore,
		},
	}

	return c.filters, nil
}

func (c *components) MQ() bus.MessageQueue {
	if c.mq != nil {
		return c.mq
	}

	c.mq = ironmq.New(c.ironQueueName)

	return c.mq
}

func (c *components) OpenPollsView() (*votingViews.OpenPolls, error) {
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

func (c *components) OpenPollsHandler() (*handlers.OpenPolls, error) {
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

	eventManager := c.EventManager()

	c.openPollsHandler = handlers.NewOpenPolls(openPollsView, viewStore, eventManager)
	c.closers = append(c.closers, c.openPollsHandler)

	refreshViewTable(c.openPollsHandler)

	return c.openPollsHandler, nil
}

func (c *components) ViewStore() (views.ViewStore, error) {
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
