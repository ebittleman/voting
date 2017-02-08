package dispatcher

import (
	"errors"
	"io"
	"log"

	"github.com/ebittleman/voting/bus"
	"github.com/ebittleman/voting/eventmanager"
)

var (
	// ErrNack returned by a filter if the message should not be handled and
	// returned back to the queue
	ErrNack = errors.New("Message filtered and not handled")
	// ErrAck returned by a filter if a message won't be handled and if it should
	// not be returned back to the queue. An example of this would be for a
	// message that fails authentication.
	ErrAck = errors.New("Message filtered, but was handled")
)

// Filter message preprocessor.
type Filter interface {
	Filter(msg bus.Message) error
}

// Runnable components have concurrent main loops that can be canceled by
// the Close method and can be block by receiving on Run's returned error
// channel
type Runnable interface {
	Run() chan error
	io.Closer
}

type busDispatcher struct {
	mq           bus.MessageQueue
	eventManager eventmanager.EventManager
	filters      []Filter

	errCh  chan error
	done   chan struct{}
	closed chan struct{}
}

// NewBusDispatcher initializes a new dispatcher
func NewBusDispatcher(
	mq bus.MessageQueue,
	eventManager eventmanager.EventManager,
	filters ...Filter,
) Runnable {
	d := new(busDispatcher)

	d.mq = mq
	d.eventManager = eventManager
	d.filters = filters

	d.errCh = make(chan error)
	d.done = make(chan struct{})
	d.closed = make(chan struct{})

	return d
}

func (d *busDispatcher) Run() chan error {
	select {
	case <-d.closed:
		return d.errCh
	default:
	}

	go d.loop()
	return d.errCh
}

func (d *busDispatcher) Close() error {
	select {
	case d.done <- struct{}{}:
	case <-d.closed:
		return nil
	}

	<-d.closed
	return nil
}

func (d *busDispatcher) loop() {
	defer close(d.closed)
	defer close(d.errCh)

	for {
		// return if we have received a close signal
		select {
		case <-d.done:
			return
		default:
		}

		// grab the next message
		msg, err := d.mq.Receive()

		// if there was an error getting the message return it
		if err != nil {
			select {
			case <-d.done:
			case d.errCh <- err:
			}
			return
		}

		// if there was no message try and receive again.
		if msg == nil {
			continue
		}

		// handle the message and log any errors
		if err := d.handle(msg); err != nil {
			log.Println("Error: Dispatching msg: ", err)
		}
	}
}

func (d *busDispatcher) filter(msg bus.Message) error {
	for _, f := range d.filters {
		if err := f.Filter(msg); err != nil {
			return err
		}
	}

	return nil
}

func (d *busDispatcher) dispatch(msg bus.Message) error {
	d.eventManager.Publish(msg.Event())
	return nil
}

func (d *busDispatcher) handle(msg bus.Message) error {
	if err := d.filter(msg); err != nil {
		switch err {
		case ErrNack:
			return d.mq.Nack(msg)
		case ErrAck:
			return d.mq.Ack(msg)
		default:
			return err
		}
	}

	if err := d.dispatch(msg); err != nil {
		return err
	}

	return d.mq.Ack(msg)
}
