package singleton

import (
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/networkflow/store"
	"github.com/stackrox/rox/central/networkflow/store/badger"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once     sync.Once
	instance store.ClusterStore
)

// Singleton provides the instance of ClusterStore to use for storing and fetching stored graphs and their associated
// information.
func Singleton() store.ClusterStore {
	once.Do(func() {
		instance = badger.NewClusterStore(globaldb.GetGlobalBadgerDB())
	})
	return instance
}
