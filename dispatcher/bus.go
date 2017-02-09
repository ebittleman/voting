package dispatcher

import (
	"log"

	"github.com/ebittleman/voting/bus"
	"github.com/ebittleman/voting/eventmanager"
)

type busDispatcher struct {
	mq           bus.MessageQueue
	eventManager eventmanager.EventManager
	subscribers  []eventmanager.Subscriber
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

func (d *busDispatcher) Run(subscribers ...eventmanager.Subscriber) error {
	return <-d.RunAsync(subscribers...)
}

func (d *busDispatcher) RunAsync(subscribers ...eventmanager.Subscriber) chan error {
	select {
	case <-d.closed:
		return d.errCh
	default:
	}

	d.subscribers = subscribers
	for _, subscriber := range d.subscribers {
		subscriber.Subscribe(d.eventManager)
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
