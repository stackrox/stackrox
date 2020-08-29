package datastore

import (
	"github.com/stackrox/rox/central/compliance/datastore/internal/store"
	"github.com/stackrox/rox/central/compliance/datastore/internal/store/bolt"
	"github.com/stackrox/rox/central/compliance/datastore/internal/store/rocksdb"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	once       sync.Once
	dsInstance DataStore
)

// Singleton returns the compliance DataStore singleton.
func Singleton() DataStore {
	var dbStore store.Store
	var err error
	once.Do(func() {
		if features.ComplianceInRocksDB.Enabled() {
			dbStore = rocksdb.NewRocksdbStore(globaldb.GetRocksDB())
		} else {
			dbStore, err = bolt.NewBoltStore(globaldb.GetGlobalDB())
		}
		utils.Must(err)

		dsInstance = NewDataStore(dbStore, NewSacFilter())
	})
	return dsInstance
}
