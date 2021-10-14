package datastore

import (
	alertDataStore "github.com/stackrox/rox/central/alert/datastore"
	"github.com/stackrox/rox/central/cluster/index"
	pgIndex "github.com/stackrox/rox/central/cluster/index/postgres"
	"github.com/stackrox/rox/central/cluster/store"
	pgStore "github.com/stackrox/rox/central/cluster/store/cluster/postgres"
	clusterRocksDB "github.com/stackrox/rox/central/cluster/store/cluster/rocksdb"
	healthRocksDB "github.com/stackrox/rox/central/cluster/store/cluster_health_status/rocksdb"
	deploymentDataStore "github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/globaldb/dackbox"
	"github.com/stackrox/rox/central/globalindex"
	namespaceDataStore "github.com/stackrox/rox/central/namespace/datastore"
	networkBaselineManager "github.com/stackrox/rox/central/networkbaseline/manager"
	netEntityDataStore "github.com/stackrox/rox/central/networkgraph/entity/datastore"
	netFlowsDataStore "github.com/stackrox/rox/central/networkgraph/flow/datastore"
	nodeDataStore "github.com/stackrox/rox/central/node/globaldatastore"
	notifierProcessor "github.com/stackrox/rox/central/notifier/processor"
	podDataStore "github.com/stackrox/rox/central/pod/datastore"
	"github.com/stackrox/rox/central/ranking"
	secretDataStore "github.com/stackrox/rox/central/secret/datastore"
	"github.com/stackrox/rox/central/sensor/service/connection"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	once sync.Once

	ad DataStore
)

func initialize() {
	var clusterStorage store.ClusterStore
	var indexer index.Indexer
	var err error
	if features.PostgresPOC.Enabled() {
		clusterStorage = pgStore.New(globaldb.GetPostgresDB())
		indexer = pgIndex.NewIndexer(globaldb.GetPostgresDB())
	} else {
		clusterStorage, err = clusterRocksDB.New(globaldb.GetRocksDB())
		utils.CrashOnError(err)
		indexer = index.New(globalindex.GetGlobalTmpIndex())
	}
	clusterHealthStorage, err := healthRocksDB.New(globaldb.GetRocksDB())
	utils.CrashOnError(err)

	ad, err = New(clusterStorage,
		clusterHealthStorage,
		indexer,
		alertDataStore.Singleton(),
		namespaceDataStore.Singleton(),
		deploymentDataStore.Singleton(),
		nodeDataStore.Singleton(),
		podDataStore.Singleton(),
		secretDataStore.Singleton(),
		netFlowsDataStore.Singleton(),
		netEntityDataStore.Singleton(),
		connection.ManagerSingleton(),
		notifierProcessor.Singleton(),
		dackbox.GetGlobalDackBox(),
		ranking.ClusterRanker(),
		networkBaselineManager.Singleton())
	utils.CrashOnError(err)
}

// Singleton provides the interface for non-service external interaction.
func Singleton() DataStore {
	once.Do(initialize)
	return ad
}
