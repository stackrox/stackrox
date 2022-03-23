package singleton

import (
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/networkgraph/flow/datastore/internal/store"
	"github.com/stackrox/rox/central/networkgraph/flow/datastore/internal/store/common"
	"github.com/stackrox/rox/central/networkgraph/flow/datastore/internal/store/postgres"
	"github.com/stackrox/rox/central/networkgraph/flow/datastore/internal/store/rocksdb"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once     sync.Once
	instance store.ClusterStore
	log      = logging.LoggerForModule()
)

// Singleton provides the instance of ClusterStore to use for storing and fetching stored graphs and their associated
// information.
func Singleton() store.ClusterStore {
	once.Do(func() {
		// Todo: Am I going to need to propagate context everywhere here too?
		log.Info("SHREWS => Singleton")
		if features.PostgresDatastore.Enabled() {
			log.Info("SHREWS => Singleton -- PG")
			instance = postgres.NewClusterStore(globaldb.GetPostgres())
		} else {
			log.Info("SHREWS => Singleton -- Rocks")
			instance = rocksdb.NewClusterStore(globaldb.GetRocksDB())
			globaldb.RegisterBucket([]byte(common.GlobalPrefix), "NetworkFlow")
		}

	})
	log.Infof("SHREWS => Singleton %s", instance)

	return instance
}
