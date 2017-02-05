package json

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"sync"
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
	Load(chan json.RawMessage) error
}

// Connection simple json database
type Connection struct {
	path         string
	tables       map[string]Table
	fileProvider func(string) (io.Reader, error)
	fileCreator  func(string) (io.Writer, error)
	sync.Mutex
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
	connection.fileCreator = func(f string) (io.Writer, error) {
		file, err := os.Create(f)
		stat, _ := file.Stat()
		log.Println("Debug: ", stat.Name())
		return file, err
	}
	connection.fileProvider = func(f string) (io.Reader, error) {
		return os.Open(f)
	}

	return connection, nil
}

// SetFileProvider gives some customizability in how we load data
func (c *Connection) SetFileProvider(fileCreator func(f string) (io.Reader, error)) {
	c.fileProvider = fileCreator
}

// SetFileCreator gives some customizability in how we save data
func (c *Connection) SetFileCreator(fileCreator func(f string) (io.Writer, error)) {
	c.fileCreator = fileCreator
}

// RegisterTable adds a table implementation the the database.
func (c *Connection) RegisterTable(name string, table Table) error {
	c.Lock()
	defer c.Unlock()
	if _, ok := c.tables[name]; ok {
		return ErrTableExists
	}

	records, errCh := make(chan json.RawMessage), make(chan error, 1)
	go func() {
		defer close(records)
		defer close(errCh)

		var (
			file     io.Reader
			line     []byte
			isPrefix bool
			err      error
		)

		if file, err = c.fileProvider(path.Join(c.path, name+".json")); err != nil {
			if _, ok := err.(*os.PathError); ok {
				return
			}
			errCh <- err
			return
		}

		if closer, ok := file.(io.Closer); ok {
			defer closer.Close()
		}

		buffer := bufio.NewReader(file)
		for err == nil {
			fullLine := make([]byte, 0, cap(line))
			line, isPrefix = line[0:0], true
			for ; isPrefix; line, isPrefix, err = buffer.ReadLine() {
				if err != nil {
					break
				}
				fullLine = append(fullLine, line...)
			}

			if err != nil && err != io.EOF {
				log.Println("Error Reading json table file: ", err)
				errCh <- err
				return
			}

			if len(fullLine) < 1 {
				continue
			}

			records <- json.RawMessage(fullLine)
		}
	}()

	if err := table.Load(records); err != nil {
		return err
	}

	if err := <-errCh; err != nil {
		return err
	}

	c.tables[name] = table
	return nil
}

// Close flushes write buffer to disk and closes the file.
func (c *Connection) Close() error {
	c.Lock()
	defer c.Unlock()
	var (
		file io.Writer
		data []byte
		err  error
	)
	for name, table := range c.tables {
		if file, err = c.fileCreator(path.Join(c.path, name+".json")); err != nil {
			return nil
		}

		for record := range table.Scan() {
			if data, err = record.MarshalJSON(); err != nil {
				if closer, ok := file.(io.Closer); ok {
					closer.Close()
				}
				return err
			}

			if _, err = fmt.Fprintln(file, string(data)); err != nil {
				if closer, ok := file.(io.Closer); ok {
					closer.Close()
				}
				return err
			}
		}

		if closer, ok := file.(io.Closer); ok {
			if err = closer.Close(); err != nil {
				return err
			}
		}
	}

	return nil
}
