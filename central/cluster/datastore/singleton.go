package datastore

import (
	alertDataStore "github.com/stackrox/rox/central/alert/datastore"
	clusterPostgres "github.com/stackrox/rox/central/cluster/store/cluster/postgres"
	clusterHealthPostgres "github.com/stackrox/rox/central/cluster/store/clusterhealth/postgres"
	clusterCVEDS "github.com/stackrox/rox/central/cve/cluster/datastore"
	deploymentDataStore "github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/globaldb"
	imageIntegrationDataStore "github.com/stackrox/rox/central/imageintegration/datastore"
	namespaceDataStore "github.com/stackrox/rox/central/namespace/datastore"
	networkBaselineManager "github.com/stackrox/rox/central/networkbaseline/manager"
	netEntityDataStore "github.com/stackrox/rox/central/networkgraph/entity/datastore"
	netFlowsDataStore "github.com/stackrox/rox/central/networkgraph/flow/datastore"
	nodeDataStore "github.com/stackrox/rox/central/node/datastore"
	notifierProcessor "github.com/stackrox/rox/central/notifier/processor"
	podDataStore "github.com/stackrox/rox/central/pod/datastore"
	"github.com/stackrox/rox/central/ranking"
	roleDataStore "github.com/stackrox/rox/central/rbac/k8srole/datastore"
	roleBindingDataStore "github.com/stackrox/rox/central/rbac/k8srolebinding/datastore"
	secretDataStore "github.com/stackrox/rox/central/secret/datastore"
	"github.com/stackrox/rox/central/sensor/service/connection"
	serviceAccountDataStore "github.com/stackrox/rox/central/serviceaccount/datastore"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	once sync.Once

	ad DataStore
)

func initialize() {
	var err error

	clusterStorage := clusterPostgres.New(globaldb.GetPostgres())
	clusterHealthStorage := clusterHealthPostgres.New(globaldb.GetPostgres())
	indexer := clusterPostgres.NewIndexer(globaldb.GetPostgres())
	clusterCVEDataStore := clusterCVEDS.Singleton()

	ad, err = New(clusterStorage,
		clusterHealthStorage,
		clusterCVEDataStore,
		alertDataStore.Singleton(),
		imageIntegrationDataStore.Singleton(),
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
		ranking.ClusterRanker(),
		indexer,
		networkBaselineManager.Singleton())

	utils.CrashOnError(err)
}

// Singleton provides the interface for non-service external interaction.
func Singleton() DataStore {
	once.Do(initialize)
	return ad
}
