package datastore

import (
	"testing"

	alertDataStore "github.com/stackrox/rox/central/alert/datastore"
	clusterPostgresStore "github.com/stackrox/rox/central/cluster/store/cluster/postgres"
	clusterHealthPostgresStore "github.com/stackrox/rox/central/cluster/store/clusterhealth/postgres"
	clusterCVEDataStore "github.com/stackrox/rox/central/cve/cluster/datastore"
	deploymentDataStore "github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/hash/datastore"
	hashManager "github.com/stackrox/rox/central/hash/manager"
	imageIntegrationDataStore "github.com/stackrox/rox/central/imageintegration/datastore"
	namespaceDataStore "github.com/stackrox/rox/central/namespace/datastore"
	networkBaselineManager "github.com/stackrox/rox/central/networkbaseline/manager"
	netEntityDataStore "github.com/stackrox/rox/central/networkgraph/entity/datastore"
	netFlowsDataStore "github.com/stackrox/rox/central/networkgraph/flow/datastore"
	nodeDataStore "github.com/stackrox/rox/central/node/datastore"
	podDataStore "github.com/stackrox/rox/central/pod/datastore"
	"github.com/stackrox/rox/central/ranking"
	roleDataStore "github.com/stackrox/rox/central/rbac/k8srole/datastore"
	roleBindingDataStore "github.com/stackrox/rox/central/rbac/k8srolebinding/datastore"
	secretDataStore "github.com/stackrox/rox/central/secret/datastore"
	"github.com/stackrox/rox/central/sensor/service/connection"
	serviceAccountDataStore "github.com/stackrox/rox/central/serviceaccount/datastore"
	"github.com/stackrox/rox/pkg/postgres"
)

// GetTestPostgresDataStore provides a datastore connected to postgres for testing purposes.
func GetTestPostgresDataStore(t *testing.T, pool postgres.DB) (DataStore, error) {
	clusterdbstore := clusterPostgresStore.New(pool)
	clusterhealthdbstore := clusterHealthPostgresStore.New(pool)
	indexer := clusterPostgresStore.NewIndexer(pool)
	alertStore, err := alertDataStore.GetTestPostgresDataStore(t, pool)
	if err != nil {
		return nil, err
	}
	namespaceStore, err := namespaceDataStore.GetTestPostgresDataStore(t, pool)
	if err != nil {
		return nil, err
	}
	deploymentStore, err := deploymentDataStore.GetTestPostgresDataStore(t, pool)
	if err != nil {
		return nil, err
	}
	nodeStore := nodeDataStore.GetTestPostgresDataStore(t, pool)
	podStore, err := podDataStore.GetTestPostgresDataStore(t, pool)
	if err != nil {
		return nil, err
	}
	secretStore, err := secretDataStore.GetTestPostgresDataStore(t, pool)
	if err != nil {
		return nil, err
	}
	netFlowStore, err := netFlowsDataStore.GetTestPostgresClusterDataStore(t, pool)
	if err != nil {
		return nil, err
	}
	netEntityStore, err := netEntityDataStore.GetTestPostgresDataStore(t, pool)
	if err != nil {
		return nil, err
	}
	serviceAccountStore, err := serviceAccountDataStore.GetTestPostgresDataStore(t, pool)
	if err != nil {
		return nil, err
	}
	k8sRoleStore := roleDataStore.GetTestPostgresDataStore(t, pool)
	k8sRoleBindingStore := roleBindingDataStore.GetTestPostgresDataStore(t, pool)
	networkBaselineManager, err := networkBaselineManager.GetTestPostgresManager(t, pool)
	if err != nil {
		return nil, err
	}
	iiStore, err := imageIntegrationDataStore.GetTestPostgresDataStore(t, pool)
	if err != nil {
		return nil, err
	}
	clusterCVEStore, err := clusterCVEDataStore.GetTestPostgresDataStore(t, pool)
	if err != nil {
		return nil, err
	}

	hashStore, err := datastore.GetTestPostgresDataStore(t, pool)
	if err != nil {
		return nil, err
	}

	sensorCnxMgr := connection.NewManager(hashManager.NewManager(hashStore))
	clusterRanker := ranking.ClusterRanker()

	return New(clusterdbstore, clusterhealthdbstore, clusterCVEStore,
		alertStore, iiStore, namespaceStore, deploymentStore,
		nodeStore, podStore, secretStore, netFlowStore, netEntityStore,
		serviceAccountStore, k8sRoleStore, k8sRoleBindingStore, sensorCnxMgr, nil,
		clusterRanker, indexer, networkBaselineManager)
}
