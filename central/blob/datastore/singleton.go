package datastore

import (
	"github.com/stackrox/rox/central/blob/datastore/search"
	"github.com/stackrox/rox/central/blob/datastore/store"
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
		store := store.New(globaldb.GetPostgres())
		searcher := search.New(store)

		ds = NewDatastore(store, searcher)
	})
	return ds
}
