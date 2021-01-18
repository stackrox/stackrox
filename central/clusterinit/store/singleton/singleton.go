package singleton

import (
	"github.com/stackrox/rox/central/clusterinit/store"
	"github.com/stackrox/rox/central/clusterinit/store/rocksdb"
	"github.com/stackrox/rox/central/globaldb"
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
		var err error
		instance, err = rocksdb.NewStore(globaldb.GetRocksDB())
		utils.Must(err)
	})
	return instance
}
