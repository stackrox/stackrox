package store

import (
	"sync"

	"github.com/stackrox/rox/central/globaldb"
)

var (
	once sync.Once

	storage ClusterStore
)

// Singleton provides the instance of ClusterStore to use for storing and fetching stored graphs and their associated
// information.
func Singleton() ClusterStore {
	once.Do(func() {
		storage = NewClusterStore(globaldb.GetGlobalDB())
	})
	return storage
}
