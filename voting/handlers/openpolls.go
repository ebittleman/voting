package handlers

import (
	"encoding/json"

	jsondb "github.com/ebittleman/voting/database/json"
	"github.com/ebittleman/voting/eventmanager"
	"github.com/ebittleman/voting/views"
	"github.com/ebittleman/voting/voting"
	votingViews "github.com/ebittleman/voting/voting/views"
)

// OpenPolls handles PollOpenedEvents and writes them to the read model's
// persitance layer
type OpenPolls struct {
	view    *votingViews.OpenPolls
	table   jsondb.Table
	wrapper EventWrapper
}

// NewOpenPolls creates a new NewOpenPolls
func NewOpenPolls(
	view *votingViews.OpenPolls,
	table jsondb.Table,
	eventManager eventmanager.EventManager,
) *OpenPolls {
	openPolls := new(OpenPolls)

	openPolls.view = view
	openPolls.table = table
	openPolls.wrapper = Subscribe(openPolls, eventManager)

	return openPolls
}

// PollOpenedHandler handles PollOpened events
func (o *OpenPolls) PollOpenedHandler(event voting.PollOpened) error {
	return o.process()
}

// PollClosedHandler handles PollClosed events
func (o *OpenPolls) PollClosedHandler(event voting.PollClosed) error {
	return o.process()
}

// Close implements io.Closer, unsubscribes from EventManager and shuts down
// goroutine.
func (o *OpenPolls) Close() error {

	if err := o.wrapper.Close(); err != nil {
		return err
	}

	return nil
}

func (o *OpenPolls) process() error {
	if err := o.view.Rebuild(); err != nil {
		switch {
		case err == votingViews.ErrClosed:
			fallthrough
		case err == votingViews.ErrProcessing:
			return nil
		default:
			return err
		}
	}

	return o.save()
}

func (o *OpenPolls) save() error {
	data, err := json.Marshal(o.view.List())
	if err != nil {
		return err
	}

	msg := json.RawMessage(data)
	row := views.ViewRow{
		ID:   "OpenPolls",
		Data: &msg,
	}

	return o.table.Put(row)
}
