package datastore

import (
	pgStore "github.com/stackrox/rox/central/customresource/datastore/internal/store/postgres"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	ds   DataStore
	once sync.Once
)

// Singleton returns the singleton custom resource datastore.
func Singleton() DataStore {
	once.Do(func() {
		db := globaldb.GetPostgres()
		ds = newDataStore(pgStore.New(db), db)
	})
	return ds
}
