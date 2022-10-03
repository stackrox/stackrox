package datastore

import (
	store "github.com/stackrox/rox/central/complianceoperator/checkresults/store"
	"github.com/stackrox/rox/central/complianceoperator/checkresults/store/postgres"
	"github.com/stackrox/rox/central/complianceoperator/checkresults/store/rocksdb"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	once sync.Once
	ds   DataStore
)

// Singleton returns the singleton datastore
func Singleton() DataStore {
	once.Do(func() {
		var storage store.Store
		if env.PostgresDatastoreEnabled.BooleanSetting() {
			storage = postgres.New(globaldb.GetPostgres())
		} else {
			var err error
			storage, err = rocksdb.New(globaldb.GetRocksDB())
			utils.Must(err)
		}
		ds = NewDatastore(storage)
	})
	return ds
}
