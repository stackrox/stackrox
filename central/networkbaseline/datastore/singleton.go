package datastore

import (
	"github.com/stackrox/stackrox/central/globaldb"
	"github.com/stackrox/stackrox/central/networkbaseline/store"
	"github.com/stackrox/stackrox/central/networkbaseline/store/postgres"
	"github.com/stackrox/stackrox/central/networkbaseline/store/rocksdb"
	"github.com/stackrox/stackrox/pkg/features"
	"github.com/stackrox/stackrox/pkg/sync"
)

var (
	once sync.Once
	ds   DataStore
)

// Singleton provides the instance of NetworkBaselineDataStore to use.
func Singleton() DataStore {
	once.Do(func() {
		var storage store.Store
		if features.PostgresDatastore.Enabled() {
			storage = postgres.New(globaldb.GetPostgres())
		} else {
			storage = rocksdb.New(globaldb.GetRocksDB())
		}
		ds = newNetworkBaselineDataStore(storage)
	})
	return ds
}
