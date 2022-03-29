package singleton

import (
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/networkgraph/flow/datastore/internal/store"
	"github.com/stackrox/rox/central/networkgraph/flow/datastore/internal/store/common"
	"github.com/stackrox/rox/central/networkgraph/flow/datastore/internal/store/postgres"
	"github.com/stackrox/rox/central/networkgraph/flow/datastore/internal/store/rocksdb"
	"github.com/stackrox/rox/pkg/features"
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
		if features.PostgresDatastore.Enabled() {
			instance = postgres.NewClusterStore(globaldb.GetPostgres())
		} else {
			instance = rocksdb.NewClusterStore(globaldb.GetRocksDB())
			globaldb.RegisterBucket([]byte(common.GlobalPrefix), "NetworkFlow")
		}

	})

	return instance
}
