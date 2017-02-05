package views

import (
	"log"
	"sort"
	"sync"

	"github.com/ebittleman/voting/eventmanager"
	"github.com/ebittleman/voting/eventstore"
	"github.com/ebittleman/voting/voting"
	"github.com/ebittleman/voting/voting/handlers"
)

// OpenPolls keeps a cache of the current open polls
type OpenPolls struct {
	ids []string

	eventStore eventstore.EventStore
	wrapper    handlers.EventWrapper

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
				if errCh != nil {
					errCh <- err
				}
				continue
			}

			pollClosedEvents, err := o.eventStore.QueryByEventType("PollClosed")
			if err != nil {
				log.Println("Warn: OpenPolls QueryBy PollClosed: ", err)
				if errCh != nil {
					errCh <- err
				}
				continue
			}

			events := append(pollOpenedEvents, pollClosedEvents...)
			sort.Sort(events)

			tmp := make([]string, 0)
			for _, event := range events {
				switch event.Type {
				case "PollOpened":
					for _, id := range tmp {
						if id == event.ID {
							continue
						}
					}
					tmp = append(tmp, event.ID)
				case "PollClosed":
					for i, id := range tmp {
						if id == event.ID {
							tmp[i] = tmp[len(tmp)-1]
							tmp[len(tmp)-1] = ""
							tmp = tmp[:len(tmp)-1]
						}
					}
				}
			}

			o.Lock()
			o.ids = tmp
			o.Unlock()
			if errCh != nil {
				close(errCh)
			}
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

	o.Lock()
	defer o.Unlock()
	if err := o.wrapper.Close(); err != nil {
		return err
	}

	select {
	case o.close <- struct{}{}:
	case <-o.done:
	}
	return nil
}

// List returns list of currently open polls
func (o *OpenPolls) List() []string {
	o.RLock()
	defer o.RUnlock()

	dst := make([]string, len(o.ids))
	copy(dst, o.ids)

	return dst
}

// PollOpenedHandler handles PollOpened events
func (o *OpenPolls) PollOpenedHandler(event voting.PollOpened) error {
	log.Println("Info: OpenPolls: Handle Poll Opened")
	select {
	case o.rebuild <- nil:
	case <-o.done:
	default:
	}
	return nil
}

// PollClosedHandler handles PollClosed events
func (o *OpenPolls) PollClosedHandler(event voting.PollClosed) error {
	log.Println("Info: OpenPolls: Handle Poll Closed")
	select {
	case o.rebuild <- nil:
	case <-o.done:
	default:
	}
	return nil
}

// NewOpenPolls creates a new NewOpenPolls
func NewOpenPolls(
	eventStore eventstore.EventStore,
	eventManager eventmanager.EventManager,
) (*OpenPolls, error) {
	openPolls := new(OpenPolls)

	openPolls.rebuild = make(chan chan error)
	openPolls.close = make(chan struct{})
	openPolls.done = make(chan struct{})

	openPolls.wrapper = handlers.Subscribe(openPolls, eventManager)
	openPolls.eventStore = eventStore

	go openPolls.loop()

	rebuilt := make(chan error)
	openPolls.rebuild <- rebuilt
	if err := <-rebuilt; err != nil {
		return nil, err
	}

	return openPolls, nil
}
