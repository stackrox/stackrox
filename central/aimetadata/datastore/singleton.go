package datastore

import (
	pgStore "github.com/stackrox/rox/central/aimetadata/datastore/internal/store/postgres"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	ds   DataStore
	once sync.Once
)

// Singleton returns the singleton AI metadata datastore.
func Singleton() DataStore {
	once.Do(func() {
		ds = newDataStore(pgStore.New(globaldb.GetPostgres()))
	})
	return ds
}
