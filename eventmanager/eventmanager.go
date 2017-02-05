package eventmanager

import (
	"errors"
	"io"
	"log"
	"sync"

	"github.com/ebittleman/voting/eventstore"
)

var (
	ErrUnhandledEventType = errors.New("Unhandled Event Type")
)

type EventHandler func(eventstore.Event) error
type Subscription interface{}
type EventManager interface {
	Publish(event eventstore.Event)
	Subscribe(eventType string, handler EventHandler) Subscription
	Unsubscribe(v Subscription) error
	io.Closer
}
type subscription struct {
	eventType string
	handler   EventHandler
}

type subscribeReq struct {
	eventType string
	handler   EventHandler
	resp      chan Subscription
}

type unsubscribeReq struct {
	sub  Subscription
	resp chan error
}

type eventManager struct {
	subscriptions map[string][]Subscription

	publishCh     chan eventstore.Event
	subscribeCh   chan subscribeReq
	unsubscribeCh chan unsubscribeReq

	done   chan chan error
	closed chan struct{}

	sync.WaitGroup
}

// New creates a new event manager
func New() EventManager {
	em := new(eventManager)
	em.init()
	return em
}

func (e *eventManager) init() {
	*e = eventManager{
		subscriptions: make(map[string][]Subscription),
		publishCh:     make(chan eventstore.Event),
		subscribeCh:   make(chan subscribeReq),
		unsubscribeCh: make(chan unsubscribeReq),
		done:          make(chan chan error),
		closed:        make(chan struct{}),
	}

	go e.loop()
}

func (e *eventManager) loop() {
	for {
		select {
		case event := <-e.publishCh:
			e.publish(event)
		case req := <-e.subscribeCh:
			req.resp <- e.subscribe(req.eventType, req.handler)
		case req := <-e.unsubscribeCh:
			e.unsubscribe(req.sub)
			req.resp <- nil
		case errCh := <-e.done:
			e.Wait()
			errCh <- nil
			close(e.closed)
			return
		}
	}
}

func (e *eventManager) publish(event eventstore.Event) {
	subs, ok := e.subscriptions[event.Type]
	if !ok {
		return
	}

	for _, sub := range subs {
		e.Add(1)
		go func(sub *subscription) {
			defer e.Done()
			if err := sub.handler(event); err != nil {
				log.Println("Error: ", err)
				go e.Unsubscribe(sub)
			}
		}(sub.(*subscription))
	}
}

func (e *eventManager) subscribe(eventType string, handler EventHandler) Subscription {
	sub := new(subscription)
	sub.eventType = eventType
	sub.handler = handler

	subscriptions, _ := e.subscriptions[eventType]
	e.subscriptions[eventType] = append(subscriptions, sub)

	return sub
}

func (e *eventManager) unsubscribe(v Subscription) {
	sub, ok := v.(*subscription)
	if !ok {
		return
	}

	eventType := sub.eventType
	subscriptions, ok := e.subscriptions[eventType]
	if !ok {
		return
	}

	for i, v := range subscriptions {
		currenSubscription, _ := v.(*subscription)
		if currenSubscription == sub {
			subscriptions[i] = subscriptions[len(subscriptions)-1]
			subscriptions[len(subscriptions)-1] = nil
			e.subscriptions[eventType] = subscriptions[:len(subscriptions)-1]
			return
		}
	}

	log.Println("Debug: Subscription Not Found")
}

func (e *eventManager) Publish(event eventstore.Event) {
	go func() {
		select {
		case e.publishCh <- event:
		case <-e.closed:
		}
	}()
}

func (e *eventManager) Subscribe(
	eventType string,
	handler EventHandler,
) Subscription {
	req := subscribeReq{
		eventType: eventType,
		handler:   handler,
		resp:      make(chan Subscription),
	}

	go func() {
		select {
		case e.subscribeCh <- req:
		case <-e.closed:
			close(req.resp)
		}
	}()

	return <-req.resp
}

func (e *eventManager) Unsubscribe(v Subscription) error {
	req := unsubscribeReq{
		sub:  v,
		resp: make(chan error),
	}

	go func() {
		select {
		case e.unsubscribeCh <- req:
		case <-e.closed:
			close(req.resp)
		}
	}()

	return <-req.resp
}

func (e *eventManager) Close() error {
	errCh := make(chan error)

	go func() {
		select {
		case e.done <- errCh:
		case <-e.closed:
			close(errCh)
		}
	}()

	return <-errCh
}
