package datastore

import (
	"github.com/stackrox/rox/central/compliance/datastore/internal/store"
	"github.com/stackrox/rox/central/compliance/datastore/internal/store/rocksdb"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once       sync.Once
	dsInstance DataStore
)

// Singleton returns the compliance DataStore singleton.
func Singleton() DataStore {
	var dbStore store.Store
	once.Do(func() {
		dbStore = rocksdb.NewRocksdbStore(globaldb.GetRocksDB())
		dsInstance = NewDataStore(dbStore, NewSacFilter())
	})
	return dsInstance
}
