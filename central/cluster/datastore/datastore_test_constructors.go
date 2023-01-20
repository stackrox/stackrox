package datastore

import (
	"testing"

	"github.com/blevesearch/bleve"
	"github.com/jackc/pgx/v4/pgxpool"
	alertDataStore "github.com/stackrox/rox/central/alert/datastore"
	"github.com/stackrox/rox/central/cluster/index"
	clusterPostgresStore "github.com/stackrox/rox/central/cluster/store/cluster/postgres"
	clusterRocksDBStore "github.com/stackrox/rox/central/cluster/store/cluster/rocksdb"
	clusterHealthPostgresStore "github.com/stackrox/rox/central/cluster/store/clusterhealth/postgres"
	clusterHealthRocksDBStore "github.com/stackrox/rox/central/cluster/store/clusterhealth/rocksdb"
	clusterCVEDataStore "github.com/stackrox/rox/central/cve/cluster/datastore"
	deploymentDataStore "github.com/stackrox/rox/central/deployment/datastore"
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
	"github.com/stackrox/rox/pkg/dackbox"
	dackboxConcurrency "github.com/stackrox/rox/pkg/dackbox/concurrency"
	rocksdbBase "github.com/stackrox/rox/pkg/rocksdb"
	"go.etcd.io/bbolt"
)

// GetTestPostgresDataStore provides a datastore connected to postgres for testing purposes.
func GetTestPostgresDataStore(t *testing.T, pool *pgxpool.Pool) (DataStore, error) {
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
	nodeStore, err := nodeDataStore.GetTestPostgresDataStore(t, pool)
	if err != nil {
		return nil, err
	}
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
	k8sRoleStore, err := roleDataStore.GetTestPostgresDataStore(t, pool)
	if err != nil {
		return nil, err
	}
	k8sRoleBindingStore, err := roleBindingDataStore.GetTestPostgresDataStore(t, pool)
	if err != nil {
		return nil, err
	}
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

	sensorCnxMgr := connection.ManagerSingleton()
	clusterRanker := ranking.ClusterRanker()

	return New(clusterdbstore, clusterhealthdbstore, clusterCVEStore,
		alertStore, iiStore, namespaceStore, deploymentStore,
		nodeStore, podStore, secretStore, netFlowStore, netEntityStore,
		serviceAccountStore, k8sRoleStore, k8sRoleBindingStore, sensorCnxMgr, nil,
		nil, clusterRanker, indexer, networkBaselineManager)
}

// GetTestRocksBleveDataStore provides a datastore connected to rocksdb and bleve for testing purposes.
func GetTestRocksBleveDataStore(t *testing.T, rocksengine *rocksdbBase.RocksDB, bleveIndex bleve.Index, dacky *dackbox.DackBox, keyFence dackboxConcurrency.KeyFence, boltengine *bbolt.DB) (DataStore, error) {
	clusterdbstore, err := clusterRocksDBStore.New(rocksengine)
	if err != nil {
		return nil, err
	}
	clusterhealthdbstore, err := clusterHealthRocksDBStore.New(rocksengine)
	if err != nil {
		return nil, err
	}
	indexer := index.New(bleveIndex)
	alertStore, err := alertDataStore.GetTestRocksBleveDataStore(t, rocksengine, bleveIndex)
	if err != nil {
		return nil, err
	}
	namespaceStore, err := namespaceDataStore.GetTestRocksBleveDataStore(t, rocksengine, bleveIndex, dacky, keyFence)
	if err != nil {
		return nil, err
	}
	deploymentStore, err := deploymentDataStore.GetTestRocksBleveDataStore(t, rocksengine, bleveIndex, dacky, keyFence)
	if err != nil {
		return nil, err
	}
	nodeStore, err := nodeDataStore.GetTestRocksBleveDataStore(t, rocksengine, bleveIndex, dacky, keyFence)
	if err != nil {
		return nil, err
	}
	podStore, err := podDataStore.GetTestRocksBleveDataStore(t, rocksengine, bleveIndex)
	if err != nil {
		return nil, err
	}
	secretStore, err := secretDataStore.GetTestRocksBleveDataStore(t, rocksengine, bleveIndex)
	if err != nil {
		return nil, err
	}
	netFlowStore, err := netFlowsDataStore.GetTestRocksBleveClusterDataStore(t, rocksengine)
	if err != nil {
		return nil, err
	}
	netEntityStore, err := netEntityDataStore.GetTestRocksBleveDataStore(t, rocksengine)
	if err != nil {
		return nil, err
	}
	serviceAccountStore, err := serviceAccountDataStore.GetTestRocksBleveDataStore(t, rocksengine, bleveIndex)
	if err != nil {
		return nil, err
	}
	k8sRoleStore, err := roleDataStore.GetTestRocksBleveDataStore(t, rocksengine, bleveIndex)
	if err != nil {
		return nil, err
	}
	k8sRoleBindingStore, err := roleBindingDataStore.GetTestRocksBleveDataStore(t, rocksengine, bleveIndex)
	if err != nil {
		return nil, err
	}
	networkBaselineManager, err := networkBaselineManager.GetTestRocksBleveManager(t, rocksengine, bleveIndex, dacky, keyFence, boltengine)
	if err != nil {
		return nil, err
	}
	iiStore, err := imageIntegrationDataStore.GetTestRocksBleveDataStore(t, boltengine, bleveIndex)
	if err != nil {
		return nil, err
	}
	sensorCnxMgr := connection.ManagerSingleton()
	clusterRanker := ranking.ClusterRanker()

	return New(clusterdbstore, clusterhealthdbstore, nil,
		alertStore, iiStore, namespaceStore, deploymentStore,
		nodeStore, podStore, secretStore, netFlowStore, netEntityStore,
		serviceAccountStore, k8sRoleStore, k8sRoleBindingStore, sensorCnxMgr, nil,
		dacky, clusterRanker, indexer, networkBaselineManager)
}
