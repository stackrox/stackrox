package datastore

import (
	"github.com/stackrox/rox/central/compliance/datastore/internal/store"
	"github.com/stackrox/rox/central/compliance/datastore/internal/store/postgres"
	"github.com/stackrox/rox/central/compliance/datastore/internal/store/rocksdb"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once       sync.Once
	dsInstance DataStore
)

// Singleton returns the compliance DataStore singleton.
func Singleton() DataStore {
	once.Do(func() {
		var dbStore store.Store
		if features.PostgresDatastore.Enabled() {
			dbStore = postgres.NewStore(globaldb.GetPostgres())
		} else {
			dbStore = rocksdb.NewRocksdbStore(globaldb.GetRocksDB())
		}
		dsInstance = NewDataStore(dbStore, NewSacFilter())
	})
	return dsInstance
}
