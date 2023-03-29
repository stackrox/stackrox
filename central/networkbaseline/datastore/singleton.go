package datastore

import (
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/networkbaseline/store"
	pgStore "github.com/stackrox/rox/central/networkbaseline/store/postgres"
	"github.com/stackrox/rox/central/networkbaseline/store/rocksdb"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once
	ds   DataStore
)

// Singleton provides the instance of NetworkBaselineDataStore to use.
func Singleton() DataStore {
	once.Do(func() {
		var storage store.Store
		if env.PostgresDatastoreEnabled.BooleanSetting() {
			storage = pgStore.New(globaldb.GetPostgres())
		} else {
			storage = rocksdb.New(globaldb.GetRocksDB())
		}
		ds = newNetworkBaselineDataStore(storage)
	})
	return ds
}
