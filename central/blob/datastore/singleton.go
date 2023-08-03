package datastore

import (
	"github.com/stackrox/rox/central/blob/datastore/search"
	"github.com/stackrox/rox/central/blob/datastore/store"
	"github.com/stackrox/rox/central/blob/datastore/store/postgres"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	ds Datastore
)

// Singleton returns the blob datastore
func Singleton() Datastore {
	once.Do(func() {
		indexer := postgres.NewIndexer(globaldb.GetPostgres())
		store := store.New(globaldb.GetPostgres())
		searcher := search.New(store, indexer)

		ds = NewDatastore(store, searcher)
	})
	return ds
}
