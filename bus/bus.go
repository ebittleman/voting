package bus

import (
	"encoding/json"

	"github.com/ebittleman/voting/eventstore"
)

// Header simple dictionary of string keys string values for housing message
// meta0data
type Header map[string]string

// Envelope wraps a before sending it across the wire
type Envelope struct {
	Header Header           `json:"header,omitempty"`
	Body   *json.RawMessage `json:"body,omitempty"`
}

// Message is passed back to recieved and expected to go back to Ack or Nak
type Message interface {
	Event() eventstore.Event
	Header() Header
}

// MessageQueue send and retreive events ove a message queue
type MessageQueue interface {
	Receive() (Message, error)
	Ack(Message) error
	Nack(Message) error
	Send(eventstore.Event) error
}
