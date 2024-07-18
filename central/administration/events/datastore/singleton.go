package datastore

import (
	"github.com/stackrox/rox/central/administration/events/datastore/internal/search"
	pgStore "github.com/stackrox/rox/central/administration/events/datastore/internal/store/postgres"
	"github.com/stackrox/rox/central/administration/events/datastore/internal/writer"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	ds DataStore
)

// Singleton returns a datastore instance to handle events.
func Singleton() DataStore {
	once.Do(func() {
		store := pgStore.New(globaldb.GetPostgres())
		searcher := search.New(store)
		writer := writer.New(store)
		ds = newDataStore(searcher, store, writer)
	})
	return ds
}
