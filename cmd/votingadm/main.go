package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/ebittleman/voting/eventstore"
	couchdb "github.com/fjl/go-couchdb"
)

func main() {
	name := "voting-1486696777"
	client, err := client()
	if err != nil {
		log.Fatal(err)
	}

	db, err := client.EnsureDB(name)
	if err != nil {
		log.Fatal(err)
	}

	if err = installView(db); err != nil {
		log.Println(err)
	}

	raw := json.RawMessage(`{"name": "dinner poll"}`)
	event := eventstore.Event{
		ID:        "myEvent",
		Version:   2,
		Type:      "PollOpened",
		Data:      &raw,
		Timestamp: time.Now().UTC().Unix(),
	}
	if _, err = db.Put(
		fmt.Sprintf("%s-%d", event.ID, event.Version),
		&event, "",
	); err != nil {
		log.Fatal(err)
	}

}

func client() (*couchdb.Client, error) {
	url := os.Getenv("COUCHDB_URL")
	client, err := couchdb.NewClient(url, http.DefaultTransport)
	if err != nil {
		return nil, err
	}

	if err = client.Ping(); err != nil {
		return nil, err
	}

	return client, nil
}

func createDatabase(client *couchdb.Client, name string) (*couchdb.DB, error) {
	return client.EnsureDB(name)
}

func installView(db *couchdb.DB) error {
	rev, err := db.Put("_design/indexes", indexDesignDoc, "")
	if err != nil {
		return err
	}

	log.Println(rev)
	return nil
}

func newDBName(name string) string {
	return fmt.Sprintf("%s-%d", name, time.Now().UTC().Unix())
}

var indexDesignDoc = struct {
	Views map[string]map[string]string `json:"views"`
}{
	Views: map[string]map[string]string{
		"events": map[string]string{
			"map": "function(doc) {\n    emit([doc.id, doc.version], null)\n}",
		},
		"by_type": map[string]string{
			"map": "function(doc) {\n  emit([doc.type, doc.id, doc.version], null)\n}",
		},
	},
}
