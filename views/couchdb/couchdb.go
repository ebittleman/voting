package couchdb

import (
	"log"

	"github.com/ebittleman/voting/views"
	couchdb "github.com/fjl/go-couchdb"
)

type couchDoc struct {
	CouchID string `json:"_id"`
	Rev     string `json:"_rev"`
	views.ViewRow
}

type store struct {
	client *couchdb.Client
	db     *couchdb.DB
}

// NewStore initializes a new jsondb backed ViewStore
func NewStore(client *couchdb.Client) (views.ViewStore, error) {
	store := new(store)
	store.client = client

	store.db = client.DB("views")

	return store, nil
}

func (s *store) Get(id string) (views.ViewRow, error) {
	doc := new(couchDoc)
	err := s.db.Get(id, &doc, nil)
	return doc.ViewRow, err
}

func (s *store) Put(row views.ViewRow) error {
	var rev string
	doc := new(couchDoc)
	if err := s.db.Get(row.ID, doc, nil); err == nil {
		rev = doc.Rev
		log.Println("CouchDB View Store REV: ", rev)
	} else {
		log.Println("CouchDB View Store: ", err)
	}

	_, err := s.db.Put(row.ID, row, rev)
	return err
}
