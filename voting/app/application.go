package app

import (
	"io"

	"github.com/ebittleman/voting/dispatcher"
	"github.com/ebittleman/voting/eventmanager"
)

// Application interface to get you started building an application.
type Application interface {
	Dispatcher() (dispatcher.Runnable, error)
	Subscribers() ([]eventmanager.Subscriber, error)
	io.Closer
}
