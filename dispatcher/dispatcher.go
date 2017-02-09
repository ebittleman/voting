package dispatcher

import (
	"errors"
	"io"

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
	Run(subscribers ...eventmanager.Subscriber) error
	RunAsync(subscribers ...eventmanager.Subscriber) chan error
	io.Closer
}
