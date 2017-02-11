package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	couchdb "github.com/fjl/go-couchdb"
)

func main() {
	name := "events"
	client, err := client()
	if err != nil {
		log.Fatal(err)
	}

	_, err = client.EnsureDB("views")
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
