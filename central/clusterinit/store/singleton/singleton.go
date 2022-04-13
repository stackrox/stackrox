package singleton

import (
	"github.com/stackrox/stackrox/central/clusterinit/store"
	"github.com/stackrox/stackrox/central/clusterinit/store/rocksdb"
	"github.com/stackrox/stackrox/central/globaldb"
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
		var err error
		instance, err = rocksdb.NewStore(globaldb.GetRocksDB())
		utils.CrashOnError(err)
	})
	return instance
}
