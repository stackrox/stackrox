package datastore

import (
	alertDataStore "github.com/stackrox/stackrox/central/alert/datastore"
	"github.com/stackrox/stackrox/central/cluster/index"
	clusterStore "github.com/stackrox/stackrox/central/cluster/store/cluster"
	clusterPostgres "github.com/stackrox/stackrox/central/cluster/store/cluster/postgres"
	clusterRocksDB "github.com/stackrox/stackrox/central/cluster/store/cluster/rocksdb"
	clusterHealthStatusStore "github.com/stackrox/stackrox/central/cluster/store/clusterhealth"
	clusterHealthPostgres "github.com/stackrox/stackrox/central/cluster/store/clusterhealth/postgres"
	healthRocksDB "github.com/stackrox/stackrox/central/cluster/store/clusterhealth/rocksdb"
	deploymentDataStore "github.com/stackrox/stackrox/central/deployment/datastore"
	"github.com/stackrox/stackrox/central/globaldb"
	"github.com/stackrox/stackrox/central/globaldb/dackbox"
	"github.com/stackrox/stackrox/central/globalindex"
	namespaceDataStore "github.com/stackrox/stackrox/central/namespace/datastore"
	networkBaselineManager "github.com/stackrox/stackrox/central/networkbaseline/manager"
	netEntityDataStore "github.com/stackrox/stackrox/central/networkgraph/entity/datastore"
	netFlowsDataStore "github.com/stackrox/stackrox/central/networkgraph/flow/datastore"
	nodeDataStore "github.com/stackrox/stackrox/central/node/globaldatastore"
	notifierProcessor "github.com/stackrox/stackrox/central/notifier/processor"
	podDataStore "github.com/stackrox/stackrox/central/pod/datastore"
	"github.com/stackrox/stackrox/central/ranking"
	roleDataStore "github.com/stackrox/stackrox/central/rbac/k8srole/datastore"
	roleBindingDataStore "github.com/stackrox/stackrox/central/rbac/k8srolebinding/datastore"
	secretDataStore "github.com/stackrox/stackrox/central/secret/datastore"
	"github.com/stackrox/stackrox/central/sensor/service/connection"
	serviceAccountDataStore "github.com/stackrox/stackrox/central/serviceaccount/datastore"
	"github.com/stackrox/stackrox/pkg/features"
	"github.com/stackrox/stackrox/pkg/sync"
	"github.com/stackrox/stackrox/pkg/utils"
)

var (
	once sync.Once

	ad DataStore
)

func initialize() {
	var clusterStorage clusterStore.Store
	var clusterHealthStorage clusterHealthStatusStore.Store
	var indexer index.Indexer
	var err error

	if features.PostgresDatastore.Enabled() {
		clusterStorage = clusterPostgres.New(globaldb.GetPostgres())
		clusterHealthStorage = clusterHealthPostgres.New(globaldb.GetPostgres())
		indexer = clusterPostgres.NewIndexer(globaldb.GetPostgres())
	} else {
		clusterStorage, err = clusterRocksDB.New(globaldb.GetRocksDB())
		utils.CrashOnError(err)
		clusterHealthStorage, err = healthRocksDB.New(globaldb.GetRocksDB())
		utils.CrashOnError(err)
		indexer = index.New(globalindex.GetGlobalTmpIndex())
	}

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
		serviceAccountDataStore.Singleton(),
		roleDataStore.Singleton(),
		roleBindingDataStore.Singleton(),
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
