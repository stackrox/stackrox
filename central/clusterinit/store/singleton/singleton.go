package singleton

import (
	"github.com/stackrox/rox/central/clusterinit/store"
	pgStore "github.com/stackrox/rox/central/clusterinit/store/postgres"
	"github.com/stackrox/rox/central/clusterinit/store/rocksdb"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	instance     store.Store
	instanceInit sync.Once
)

// Singleton returns the singleton data store for cluster init bundles.
func Singleton() store.Store {
	instanceInit.Do(func() {
		var underlying store.UnderlyingStore
		if env.PostgresDatastoreEnabled.BooleanSetting() {
			underlying = pgStore.New(globaldb.GetPostgres())
		} else {
			var err error
			underlying, err = rocksdb.New(globaldb.GetRocksDB())
			utils.CrashOnError(err)
		}
		instance = store.NewStore(underlying)
	})
	return instance
}
