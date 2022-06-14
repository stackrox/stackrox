package singleton

import (
	"github.com/stackrox/stackrox/central/clusterinit/store"
	"github.com/stackrox/stackrox/central/clusterinit/store/postgres"
	"github.com/stackrox/stackrox/central/clusterinit/store/rocksdb"
	"github.com/stackrox/stackrox/central/globaldb"
	"github.com/stackrox/stackrox/pkg/features"
	"github.com/stackrox/stackrox/pkg/sync"
	"github.com/stackrox/stackrox/pkg/utils"
)

var (
	instance     store.Store
	instanceInit sync.Once
)

// Singleton returns the singleton data store for cluster init bundles.
func Singleton() store.Store {
	instanceInit.Do(func() {
		var underlying store.UnderlyingStore
		if features.PostgresDatastore.Enabled() {
			underlying = postgres.New(globaldb.GetPostgres())
		} else {
			var err error
			underlying, err = rocksdb.New(globaldb.GetRocksDB())
			utils.CrashOnError(err)
		}
		instance = store.NewStore(underlying)
	})
	return instance
}
