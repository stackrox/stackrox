package store

import (
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	storeInstance     Store
	storeInstanceInit sync.Once
)

// Singleton returns the singleton instance for the sensor connection manager.
func Singleton() Store {
	storeInstanceInit.Do(func() {
		storeInstance = newStore(globaldb.GetGlobalDB())
	})

	return storeInstance
}
