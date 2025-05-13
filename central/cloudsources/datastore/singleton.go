package datastore

import (
	"github.com/stackrox/rox/central/cloudsources/datastore/internal/search"
	pgStore "github.com/stackrox/rox/central/cloudsources/datastore/internal/store/postgres"
	discoveredClustersDS "github.com/stackrox/rox/central/discoveredclusters/datastore"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	ds   DataStore
	once sync.Once
)

// Singleton returns the singleton providing access to the cloud sources store.
func Singleton() DataStore {
	once.Do(func() {
		store := pgStore.New(globaldb.GetPostgres())
		searcher := search.New(store)
		ds = newDataStore(searcher, store, discoveredClustersDS.Singleton())
	})
	return ds
}
