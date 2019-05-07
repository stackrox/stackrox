package globaldatastore

import (
	"github.com/stackrox/rox/central/globalindex"
	"github.com/stackrox/rox/central/node/globalstore"
	"github.com/stackrox/rox/central/node/index"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	globalDataStoreInstance     GlobalDataStore
	initGlobalDataStoreInstance sync.Once
)

// Singleton returns the singleton global node datastore instance.
func Singleton() GlobalDataStore {
	initGlobalDataStoreInstance.Do(func() {
		var err error
		indexer := index.New(globalindex.GetGlobalIndex())
		globalDataStoreInstance, err = New(globalstore.Singleton(), indexer)
		utils.Must(err)
	})
	return globalDataStoreInstance
}
