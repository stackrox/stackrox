package datastore

import (
	"testing"

	alertDataStore "github.com/stackrox/rox/central/alert/datastore"
	clusterPostgresStore "github.com/stackrox/rox/central/cluster/store/cluster/postgres"
	clusterHealthPostgresStore "github.com/stackrox/rox/central/cluster/store/clusterhealth/postgres"
	compliancePruning "github.com/stackrox/rox/central/complianceoperator/v2/pruner"
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
func GetTestPostgresDataStore(t testing.TB, pool postgres.DB) (DataStore, error) {
	clusterdbstore := clusterPostgresStore.New(pool)
	clusterhealthdbstore := clusterHealthPostgresStore.New(pool)
	alertStore := alertDataStore.GetTestPostgresDataStore(t, pool)
	namespaceStore, err := namespaceDataStore.GetTestPostgresDataStore(t, pool)
	if err != nil {
		return nil, err
	}
	deploymentStore, err := deploymentDataStore.GetTestPostgresDataStore(t, pool)
	if err != nil {
		return nil, err
	}
	nodeStore := nodeDataStore.GetTestPostgresDataStore(t, pool)
	podStore := podDataStore.GetTestPostgresDataStore(t, pool)
	secretStore := secretDataStore.GetTestPostgresDataStore(t, pool)
	netFlowStore, err := netFlowsDataStore.GetTestPostgresClusterDataStore(t, pool)
	if err != nil {
		return nil, err
	}
	netEntityStore := netEntityDataStore.GetTestPostgresDataStore(t, pool)
	serviceAccountStore := serviceAccountDataStore.GetTestPostgresDataStore(t, pool)
	k8sRoleStore := roleDataStore.GetTestPostgresDataStore(t, pool)
	k8sRoleBindingStore := roleBindingDataStore.GetTestPostgresDataStore(t, pool)
	networkBaselineManager, err := networkBaselineManager.GetTestPostgresManager(t, pool)
	if err != nil {
		return nil, err
	}
	iiStore := imageIntegrationDataStore.GetTestPostgresDataStore(t, pool)
	clusterCVEStore, err := clusterCVEDataStore.GetTestPostgresDataStore(t, pool)
	if err != nil {
		return nil, err
	}

	hashStore := datastore.GetTestPostgresDataStore(t, pool)

	sensorCnxMgr := connection.NewManager(hashManager.NewManager(hashStore))
	clusterRanker := ranking.ClusterRanker()

	compliancePruner := compliancePruning.GetTestPruner(t, pool)

	return New(clusterdbstore, clusterhealthdbstore, clusterCVEStore,
		alertStore, iiStore, namespaceStore, deploymentStore,
		nodeStore, podStore, secretStore, netFlowStore, netEntityStore,
		serviceAccountStore, k8sRoleStore, k8sRoleBindingStore, sensorCnxMgr, nil,
		clusterRanker, networkBaselineManager, compliancePruner)
}
