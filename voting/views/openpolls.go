package views

import (
	"errors"
	"log"
	"sort"
	"sync"

	"github.com/ebittleman/voting/eventstore"
	"github.com/ebittleman/voting/voting/model"
)

var (
	// ErrProcessing returned when a view is already rebuilding.
	ErrProcessing = errors.New("View is already processing")
	// ErrClosed returned when a view has been shutdown and is no longer servicing
	// requests.
	ErrClosed = errors.New("View is not current running")
)

type ballotStub struct {
	ID     string        `json:"id"`
	Issues []model.Issue `json:"issues"`
}

// OpenPolls keeps a cache of the current open polls
type OpenPolls struct {
	ids map[string]*ballotStub

	eventStore eventstore.EventStore

	rebuild chan chan error
	close   chan struct{}
	done    chan struct{}

	sync.RWMutex
}

func (o *OpenPolls) loop() {
	for {
		select {
		case <-o.close:
			close(o.done)
			return
		case errCh := <-o.rebuild:
			pollOpenedEvents, err := o.eventStore.QueryByEventType("PollOpened")
			if err != nil {
				log.Println("Warn: OpenPolls QueryBy PollOpened: ", err)
				errCh <- err
				continue
			}

			pollClosedEvents, err := o.eventStore.QueryByEventType("PollClosed")
			if err != nil {
				log.Println("Warn: OpenPolls QueryBy PollClosed: ", err)
				errCh <- err
				continue
			}

			events := append(pollOpenedEvents, pollClosedEvents...)
			sort.Sort(events)

			tmp := make(map[string]*ballotStub)
			for _, event := range events {
				switch event.Type {
				case "PollOpened":
					tmp[event.ID] = nil
				case "PollClosed":
					delete(tmp, event.ID)
				}
			}

			for id := range tmp {
				events, err := o.eventStore.Query(id)
				if err == nil {
					poll := model.LoadPoll(id, events)
					stub := new(ballotStub)
					stub.ID = poll.ID
					stub.Issues = poll.Issues
					tmp[id] = stub
					continue
				}
				log.Println("Warn: OpenPolls QueryBy PollClosed: ", err)
				errCh <- err
				break
			}

			o.Lock()
			o.ids = tmp
			o.Unlock()
			close(errCh)
		}
	}
}

// Close implements io.Closer, unsubscribes from EventManager and shuts down
// goroutine.
func (o *OpenPolls) Close() error {
	select {
	case <-o.done:
		return nil
	default:
	}

	select {
	case o.close <- struct{}{}:
		<-o.done
	case <-o.done:
	}
	return nil
}

// List returns list of currently open polls
func (o *OpenPolls) List() []interface{} {
	o.RLock()
	defer o.RUnlock()

	dst := make([]interface{}, 0, len(o.ids))
	for _, stub := range o.ids {
		dst = append(dst, stub)
	}

	return dst
}

// Rebuild triggers the view to rebuild
func (o *OpenPolls) Rebuild() (err error) {
	rebuilt := make(chan error)
	select {
	case o.rebuild <- rebuilt:
	case <-o.done:
		return ErrClosed
	default:
		return ErrProcessing
	}
	return <-rebuilt
}

// NewOpenPolls creates a new NewOpenPolls
func NewOpenPolls(
	eventStore eventstore.EventStore,
) (*OpenPolls, error) {
	openPolls := new(OpenPolls)

	openPolls.rebuild = make(chan chan error, 1)
	openPolls.close = make(chan struct{})
	openPolls.done = make(chan struct{})

	openPolls.eventStore = eventStore

	go openPolls.loop()

	rebuilt := make(chan error)
	openPolls.rebuild <- rebuilt
	if err := <-rebuilt; err != nil {
		return nil, err
	}

	return openPolls, nil
}
