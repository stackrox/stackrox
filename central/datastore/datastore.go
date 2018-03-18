package datastore

import (
	"bitbucket.org/stack-rox/apollo/central/db"
	"bitbucket.org/stack-rox/apollo/central/search"
)

// DataStore is a wrapper around the flow of data
type DataStore struct {
	db      db.Storage
	indexer search.Indexer
}
