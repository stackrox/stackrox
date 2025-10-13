package singleton

import (
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/networkgraph/flow/datastore/internal/store"
	pgStore "github.com/stackrox/rox/central/networkgraph/flow/datastore/internal/store/postgres"
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
		instance = pgStore.NewClusterStore(globaldb.GetPostgres())
	})

	return instance
}
