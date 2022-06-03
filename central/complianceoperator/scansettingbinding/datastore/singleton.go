package datastore

import (
	store "github.com/stackrox/rox/central/complianceoperator/scansettingbinding/store"
	"github.com/stackrox/rox/central/complianceoperator/scansettingbinding/store/postgres"
	"github.com/stackrox/rox/central/complianceoperator/scansettingbinding/store/rocksdb"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/pkg/features"
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
		if features.PostgresDatastore.Enabled() {
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
