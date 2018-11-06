package store

import (
	"sync"

	"github.com/stackrox/rox/central/globaldb"
)

var (
	globalStoreInstance     GlobalStore
	initGlobalStoreInstance sync.Once
)

// Singleton returns the singleton global node instance.
func Singleton() GlobalStore {
	initGlobalStoreInstance.Do(func() {
		globalStoreInstance = NewGlobalStore(globaldb.GetGlobalDB())
	})
	return globalStoreInstance
}
