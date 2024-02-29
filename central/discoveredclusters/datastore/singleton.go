package datastore

import (
	"github.com/stackrox/rox/central/discoveredclusters/datastore/internal/search"
	pgStore "github.com/stackrox/rox/central/discoveredclusters/datastore/internal/store/postgres"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	ds   DataStore
	once sync.Once
)

// Singleton returns the singleton providing access to the discovered clusters store.
func Singleton() DataStore {
	once.Do(func() {
		store := pgStore.New(globaldb.GetPostgres())
		searcher := search.New(store)
		ds = newDataStore(searcher, store)
	})
	return ds
}
