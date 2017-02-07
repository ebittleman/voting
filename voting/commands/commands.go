package commands

import "errors"

var (
	// ErrNotImplemented place holder error until a command is implemented
	ErrNotImplemented = errors.New("Command not implemented")
)

// Command simple interface for running a command.
type Command interface {
	Run() error
}
