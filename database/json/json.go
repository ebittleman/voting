package json

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
)

var (
	// ErrNotImplemented a function is in development
	ErrNotImplemented = errors.New("Not Implemented")
	// ErrInvalidPath returned by Open if the path is not a directory
	ErrInvalidPath = errors.New("Provided path not a directory")
	// ErrTableExists returned when attempting to register a table that already
	// exists.
	ErrTableExists = errors.New("Table Already Exists")
)

// Table implementation for marshaling and unmarshaling records
type Table interface {
	Scan() chan json.RawMessage
	Put(interface{}) error
	Load(chan json.RawMessage)
}

// Connection simple json database
type Connection struct {
	path         string
	tables       map[string]Table
	fileProvider func(string) (io.ReadCloser, error)
	fileCreator  func(string) (io.WriteCloser, error)
}

// Open creates a new json db connection from the passed file
func Open(path string) (*Connection, error) {
	if stat, err := os.Stat(path); err != nil {
		return nil, err
	} else if !stat.IsDir() {
		return nil, ErrInvalidPath
	}

	connection := new(Connection)
	connection.path = path
	connection.tables = make(map[string]Table)
	connection.fileCreator = func(f string) (io.WriteCloser, error) {
		return os.Create(f)
	}
	connection.fileProvider = func(f string) (io.ReadCloser, error) {
		return os.Open(f)
	}

	return connection, nil
}

// RegisterTable adds a table implementation the the database.
func (c *Connection) RegisterTable(name string, table Table) error {
	if _, ok := c.tables[name]; ok {
		return ErrTableExists
	}

	records, errCh := make(chan json.RawMessage), make(chan error, 1)
	go func() {
		defer close(records)
		defer close(errCh)
		var line []byte
		file, err := c.fileProvider(path.Join(c.path, name+".json"))
		if err != nil {
			if _, ok := err.(*os.PathError); ok {
				return
			}
			errCh <- err
			return
		}
		defer file.Close()
		buffer := bufio.NewReader(file)
		for err == nil {
			line, _, err = buffer.ReadLine()
			if len(line) < 1 {
				continue
			}
			records <- json.RawMessage(line)
		}
		if err != nil && err != io.EOF {
			errCh <- err
			return
		}
	}()

	table.Load(records)
	c.tables[name] = table
	return <-errCh
}

// Close flushes write buffer to disk and closes the file.
func (c *Connection) Close() error {
	var (
		file io.WriteCloser
		data []byte
		err  error
	)
	for name, table := range c.tables {
		if file, err = c.fileCreator(path.Join(c.path, name+".json")); err != nil {
			return nil
		}

		for record := range table.Scan() {
			if data, err = record.MarshalJSON(); err != nil {
				file.Close()
				return err
			}

			if _, err = fmt.Fprintln(file, string(data)); err != nil {
				file.Close()
				return err
			}
		}

		if err = file.Close(); err != nil {
			return err
		}
	}

	return nil
}