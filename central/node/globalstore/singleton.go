package globalstore

import (
	"sync"

	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/globalindex"
	"github.com/stackrox/rox/central/node/index"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	globalStoreInstance     GlobalStore
	initGlobalStoreInstance sync.Once
)

// Singleton returns the singleton global node instance.
func Singleton() GlobalStore {
	initGlobalStoreInstance.Do(func() {
		var err error
		indexer := index.New(globalindex.GetGlobalIndex())
		globalStoreInstance, err = NewGlobalStore(globaldb.GetGlobalDB(), indexer)
		utils.Must(err)
	})
	return globalStoreInstance
}
