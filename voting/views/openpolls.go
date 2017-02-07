package views

import (
	"errors"
	"log"
	"sort"
	"sync"

	"github.com/ebittleman/voting/eventstore"
)

var (
	// ErrProcessing returned when a view is already rebuilding.
	ErrProcessing = errors.New("View is already processing")
	// ErrClosed returned when a view has been shutdown and is no longer servicing
	// requests.
	ErrClosed = errors.New("View is not current running")
)

// OpenPolls keeps a cache of the current open polls
type OpenPolls struct {
	ids map[string]struct{}

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

			tmp := make(map[string]struct{}, 0)
			for _, event := range events {
				switch event.Type {
				case "PollOpened":
					tmp[event.ID] = struct{}{}
				case "PollClosed":
					delete(tmp, event.ID)
				}
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
func (o *OpenPolls) List() []string {
	o.RLock()
	defer o.RUnlock()

	dst := make([]string, 0, len(o.ids))
	for id := range o.ids {
		dst = append(dst, id)
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

	openPolls.rebuild = make(chan chan error)
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
