package datastore

import (
	store "github.com/stackrox/stackrox/central/complianceoperator/rules/store"
	"github.com/stackrox/stackrox/central/complianceoperator/rules/store/postgres"
	"github.com/stackrox/stackrox/central/complianceoperator/rules/store/rocksdb"
	"github.com/stackrox/stackrox/central/globaldb"
	"github.com/stackrox/stackrox/pkg/features"
	"github.com/stackrox/stackrox/pkg/sync"
	"github.com/stackrox/stackrox/pkg/utils"
)

var (
	once sync.Once
	ds   DataStore
)

// Singleton returns the singleton datastore
func Singleton() DataStore {
	once.Do(func() {
		var storage store.Store
		if features.PostgresDatastore.Enabled() {
			storage = postgres.New(globaldb.GetPostgres())
		} else {
			var err error
			storage, err = rocksdb.New(globaldb.GetRocksDB())
			utils.CrashOnError(err)
		}
		var err error
		ds, err = NewDatastore(storage)
		utils.CrashOnError(err)
	})
	return ds
}
