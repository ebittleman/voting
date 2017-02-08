package views

import (
	"encoding/json"
	"errors"
)

var (
	// ErrExpectedViewRow returned when an invalid type is passed to a views table
	ErrExpectedViewRow = errors.New("Expected views.ViewRow")
	// ErrRowNotFound returned when a requested view row as not found.
	ErrRowNotFound = errors.New("ViewRow not found")
)

// ViewRow holds view data
type ViewRow struct {
	ID   string           `json:"id"`
	Data *json.RawMessage `json:"data"`
}

// ViewStore implementations persist views to backing service.
type ViewStore interface {
	Get(id string) (ViewRow, error)
	Put(row ViewRow) error
}
