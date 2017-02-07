package ironmq

import (
	"encoding/json"
	"errors"
	"log"

	"github.com/ebittleman/voting/bus"
	"github.com/ebittleman/voting/eventstore"
	"github.com/iron-io/iron_go3/mq"
)

type ironMessageQueue interface {
	PushString(body string) (id string, err error)
	PushStrings(bodies ...string) (ids []string, err error)
	Peek() ([]mq.Message, error)
	PeekN(n int) ([]mq.Message, error)
	Reserve() (msg *mq.Message, err error)
	ReserveN(n int) ([]mq.Message, error)
	LongPoll(n, timeout, wait int, delete bool) ([]mq.Message, error)
}

type message struct {
	mqMsg  *mq.Message
	header bus.Header
	event  eventstore.Event
}

func (m message) Event() eventstore.Event {
	return m.event
}

func (m message) Header() bus.Header {
	return m.header
}

type messageQueue struct {
	queue ironMessageQueue
}

// New creates a new message queue bound to an ironmq message queue
func New(queueName string) bus.MessageQueue {
	mq := new(messageQueue)
	mq.queue = queueFactory(queueName)

	return mq
}

func (m *messageQueue) Receive() (bus.Message, error) {
	var (
		env    bus.Envelope
		msg    message
		mqMsgs []mq.Message
		err    error
	)

	mqMsgs, err = m.queue.LongPoll(1, 60, 30, false)
	for ; len(mqMsgs) < 1 && err == nil; mqMsgs, err = m.queue.LongPoll(1, 60, 30, false) {
		log.Println("Info: Didn't receive any messages trying again.")
	}
	if err != nil {
		return nil, err
	}

	mqMsg := mqMsgs[0]
	msg.mqMsg = &mqMsg

	if err = json.Unmarshal([]byte(mqMsg.Body), &env); err != nil {
		return nil, err
	}

	msg.header = env.Header

	envBody, err := env.Body.MarshalJSON()
	if err != nil {
		return nil, err
	}

	if err = json.Unmarshal(envBody, &msg.event); err != nil {
		return nil, err
	}

	return msg, nil
}

func (m *messageQueue) Ack(msg bus.Message) error {
	ironMsg, ok := msg.(message)
	if !ok {
		return errors.New("Invalid bus.Message")
	}

	return ironMsg.mqMsg.Delete()
}

func (m *messageQueue) Nack(msg bus.Message) error {
	ironMsg, ok := msg.(message)
	if !ok {
		return errors.New("Invalid bus.Message")
	}

	return ironMsg.mqMsg.Release(3)
}

func (m *messageQueue) Send(event eventstore.Event) error {
	env := new(bus.Envelope)
	eventData, err := json.Marshal(event)
	if err != nil {
		return err
	}

	raw := json.RawMessage(eventData)
	env.Body = &raw
	msg, err := json.Marshal(env)
	if err != nil {
		return err
	}

	_, err = m.queue.PushString(string(msg))
	if err != nil {
		return err
	}

	return nil
}

func queueFactory(queueName string) ironMessageQueue {
	return mq.New(queueName)
}
