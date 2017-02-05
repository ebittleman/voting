package json

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
)

func TestOpenWithNonExistantPath(t *testing.T) {
	conn, err := Open("xxyyzz")
	if err == nil {
		t.Fatal("Expected Error")
	}
	if _, ok := err.(*os.PathError); !ok {
		t.Fatalf("Expected: *os.PathError, Got: %T", err)
	}

	if conn != nil {
		t.Fatal("Expected conn to be nil")
	}
}

func TestOpenWithFile(t *testing.T) {
	conn, err := Open("json.go")
	if err != ErrInvalidPath {
		t.Fatal("Expected Error: ", ErrInvalidPath)
	}
	if conn != nil {
		t.Fatal("Expected conn to be nil")
	}
}

func TestOpenWithSuccess(t *testing.T) {
	path := os.TempDir()
	conn, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}

	if conn == nil {
		t.Fatal("Expected conn to be non-nil")
	}
}

func TestRegisterTableSuccessSkipLoading(t *testing.T) {
	var table mockTable
	table.t = t
	conn, err := Open(".")
	if err != nil {
		t.Fatal(err)
	}

	if err := conn.RegisterTable("test", table); err != nil {
		t.Fatalf("%T: %v", err, err)
	}
}

func TestRegisterTableFailWithDuplication(t *testing.T) {
	var table mockTable
	table.t = t
	conn, err := Open(".")
	if err != nil {
		t.Fatal(err)
	}

	if err := conn.RegisterTable("test", table); err != nil {
		t.Fatalf("%T: %v", err, err)
	}

	if err := conn.RegisterTable("test", table); err != ErrTableExists {
		t.Fatalf("Expected: %v, Got: %v", ErrTableExists, err)
	}
}

func TestRegisterTableFailedWithNonPathError(t *testing.T) {
	var table mockTable
	table.t = t
	conn, err := Open(".")
	if err != nil {
		t.Fatal(err)
	}

	conn.fileProvider = func(_ string) (io.Reader, error) {
		return nil, fmt.Errorf("Unknown, But Expected Test Error")
	}

	if err = conn.RegisterTable("test", table); err == nil {
		t.Fatal("Expected an Error")
	}

	t.Log(err)
}

func TestRegisterTableSuccessWithLoading(t *testing.T) {
	var table mockTable
	table.t = t
	conn, err := Open(".")
	if err != nil {
		t.Fatal(err)
	}

	conn.fileProvider = func(_ string) (io.Reader, error) {
		return mockFile{
			Reader: strings.NewReader(`{"key": "value"}`),
		}, nil
	}

	if err := conn.RegisterTable("test", table); err != nil {
		t.Fatalf("%T: %v", err, err)
	}
}

type mockTable struct {
	t *testing.T
}

func (m mockTable) Scan() chan json.RawMessage {
	return nil
}

func (m mockTable) Put(interface{}) error {
	return nil
}

func (m mockTable) Load(records chan json.RawMessage) error {
	for record := range records {
		m.t.Log(string(record))
	}

	return nil
}

type mockFile struct {
	io.Reader
	io.Writer
}

func (m mockFile) Close() error {
	return nil
}
