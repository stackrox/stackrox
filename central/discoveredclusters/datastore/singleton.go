package datastore

import (
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
		ds = newDataStore(pgStore.New(globaldb.GetPostgres()))
	})
	return ds
}
