package globaldatastore

import (
	"github.com/stackrox/stackrox/central/node/datastore/dackbox/datastore"
	"github.com/stackrox/stackrox/central/node/datastore/dackbox/globaldatastore"
	"github.com/stackrox/stackrox/pkg/sync"
)

var (
	globalDataStoreInstance     GlobalDataStore
	initGlobalDataStoreInstance sync.Once
)

// Singleton returns the singleton global node datastore instance.
func Singleton() GlobalDataStore {
	initGlobalDataStoreInstance.Do(func() {
		globalDataStoreInstance, _ = globaldatastore.New(datastore.Singleton())
	})
	return globalDataStoreInstance
}
