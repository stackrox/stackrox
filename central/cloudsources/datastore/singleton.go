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

// Singleton returns the singleton providing access to the external backups store.
func Singleton() DataStore {
	once.Do(func() {
		searcher := search.New(pgStore.NewIndexer(globaldb.GetPostgres()))
		store := pgStore.New(globaldb.GetPostgres())
		ds = newDataStore(searcher, store, discoveredClustersDS.Singleton())
	})
	return ds
}
