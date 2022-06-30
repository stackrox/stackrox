package datastore

import (
	alertDataStore "github.com/stackrox/rox/central/alert/datastore"
	"github.com/stackrox/rox/central/cluster/index"
	clusterStore "github.com/stackrox/rox/central/cluster/store/cluster"
	clusterPostgres "github.com/stackrox/rox/central/cluster/store/cluster/postgres"
	clusterRocksDB "github.com/stackrox/rox/central/cluster/store/cluster/rocksdb"
	clusterHealthStatusStore "github.com/stackrox/rox/central/cluster/store/clusterhealth"
	clusterHealthPostgres "github.com/stackrox/rox/central/cluster/store/clusterhealth/postgres"
	healthRocksDB "github.com/stackrox/rox/central/cluster/store/clusterhealth/rocksdb"
	deploymentDataStore "github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/globaldb"
	globalDackBox "github.com/stackrox/rox/central/globaldb/dackbox"
	"github.com/stackrox/rox/central/globalindex"
	imageIntegrationDataStore "github.com/stackrox/rox/central/imageintegration/datastore"
	namespaceDataStore "github.com/stackrox/rox/central/namespace/datastore"
	networkBaselineManager "github.com/stackrox/rox/central/networkbaseline/manager"
	netEntityDataStore "github.com/stackrox/rox/central/networkgraph/entity/datastore"
	netFlowsDataStore "github.com/stackrox/rox/central/networkgraph/flow/datastore"
	nodeDataStore "github.com/stackrox/rox/central/node/globaldatastore"
	notifierProcessor "github.com/stackrox/rox/central/notifier/processor"
	podDataStore "github.com/stackrox/rox/central/pod/datastore"
	"github.com/stackrox/rox/central/ranking"
	roleDataStore "github.com/stackrox/rox/central/rbac/k8srole/datastore"
	roleBindingDataStore "github.com/stackrox/rox/central/rbac/k8srolebinding/datastore"
	secretDataStore "github.com/stackrox/rox/central/secret/datastore"
	"github.com/stackrox/rox/central/sensor/service/connection"
	serviceAccountDataStore "github.com/stackrox/rox/central/serviceaccount/datastore"
	"github.com/stackrox/rox/pkg/dackbox"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	once sync.Once

	ad DataStore
)

func initialize() {
	var dackbox *dackbox.DackBox
	var clusterStorage clusterStore.Store
	var clusterHealthStorage clusterHealthStatusStore.Store
	var indexer index.Indexer
	var err error

	if features.PostgresDatastore.Enabled() {
		clusterStorage = clusterPostgres.New(globaldb.GetPostgres())
		clusterHealthStorage = clusterHealthPostgres.New(globaldb.GetPostgres())
		indexer = clusterPostgres.NewIndexer(globaldb.GetPostgres())
	} else {
		dackbox = globalDackBox.GetGlobalDackBox()
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
		dackbox,
		ranking.ClusterRanker(),
		networkBaselineManager.Singleton(),
		imageIntegrationDataStore.Singleton())

	utils.CrashOnError(err)
}

// Singleton provides the interface for non-service external interaction.
func Singleton() DataStore {
	once.Do(initialize)
	return ad
}
