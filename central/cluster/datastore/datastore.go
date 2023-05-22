package datastore

import (
	"context"
	"testing"

	"github.com/blevesearch/bleve"
	"github.com/jackc/pgx/v4/pgxpool"
	alertDataStore "github.com/stackrox/rox/central/alert/datastore"
	"github.com/stackrox/rox/central/cluster/datastore/internal/search"
	"github.com/stackrox/rox/central/cluster/index"
	clusterStore "github.com/stackrox/rox/central/cluster/store/cluster"
	clusterPostgresStore "github.com/stackrox/rox/central/cluster/store/cluster/postgres"
	clusterRocksDBStore "github.com/stackrox/rox/central/cluster/store/cluster/rocksdb"
	clusterHealthStore "github.com/stackrox/rox/central/cluster/store/clusterhealth"
	clusterHealthPostgresStore "github.com/stackrox/rox/central/cluster/store/clusterhealth/postgres"
	clusterHealthRocksDBStore "github.com/stackrox/rox/central/cluster/store/clusterhealth/rocksdb"
	clusterCVEDS "github.com/stackrox/rox/central/cve/cluster/datastore"
	clusterCVEDataStore "github.com/stackrox/rox/central/cve/cluster/datastore"
	deploymentDataStore "github.com/stackrox/rox/central/deployment/datastore"
	imageIntegrationDataStore "github.com/stackrox/rox/central/imageintegration/datastore"
	namespaceDataStore "github.com/stackrox/rox/central/namespace/datastore"
	networkBaselineManager "github.com/stackrox/rox/central/networkbaseline/manager"
	netEntityDataStore "github.com/stackrox/rox/central/networkgraph/entity/datastore"
	netFlowsDataStore "github.com/stackrox/rox/central/networkgraph/flow/datastore"
	nodeNodeDataStore "github.com/stackrox/rox/central/node/datastore/dackbox/datastore"
	nodeGlobalDataStore "github.com/stackrox/rox/central/node/datastore/dackbox/globaldatastore"
	nodeDataStore "github.com/stackrox/rox/central/node/globaldatastore"
	notifierProcessor "github.com/stackrox/rox/central/notifier/processor"
	podDataStore "github.com/stackrox/rox/central/pod/datastore"
	"github.com/stackrox/rox/central/ranking"
	roleDataStore "github.com/stackrox/rox/central/rbac/k8srole/datastore"
	roleBindingDataStore "github.com/stackrox/rox/central/rbac/k8srolebinding/datastore"
	"github.com/stackrox/rox/central/role/resources"
	secretDataStore "github.com/stackrox/rox/central/secret/datastore"
	"github.com/stackrox/rox/central/sensor/service/connection"
	serviceAccountDataStore "github.com/stackrox/rox/central/serviceaccount/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/dackbox"
	dackboxConcurrency "github.com/stackrox/rox/pkg/dackbox/concurrency"
	"github.com/stackrox/rox/pkg/dackbox/graph"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/logging"
	rocksdbBase "github.com/stackrox/rox/pkg/rocksdb"
	"github.com/stackrox/rox/pkg/sac"
	pkgSearch "github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/simplecache"
	"go.etcd.io/bbolt"
)

var (
	log        = logging.LoggerForModule()
	cleanupCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Node, resources.Cluster)))
)

// DataStore is the entry point for modifying Cluster data.
//go:generate mockgen-wrapper
type DataStore interface {
	GetCluster(ctx context.Context, id string) (*storage.Cluster, bool, error)
	GetClusterName(ctx context.Context, id string) (string, bool, error)
	GetClusters(ctx context.Context) ([]*storage.Cluster, error)
	CountClusters(ctx context.Context) (int, error)
	Exists(ctx context.Context, id string) (bool, error)

	AddCluster(ctx context.Context, cluster *storage.Cluster) (string, error)
	UpdateCluster(ctx context.Context, cluster *storage.Cluster) error
	RemoveCluster(ctx context.Context, id string, done *concurrency.Signal) error
	// UpdateClusterStatus updates the cluster status. Note that any cluster-upgrade or cluster-cert-expiry status
	// passed to this endpoint will be ignored. To insert that, callers MUST separately call
	// UpdateClusterUpgradeStatus or UpdateClusterCertExpiry status.
	UpdateClusterStatus(ctx context.Context, id string, status *storage.ClusterStatus) error
	UpdateClusterUpgradeStatus(ctx context.Context, id string, clusterUpgradeStatus *storage.ClusterUpgradeStatus) error
	UpdateClusterCertExpiryStatus(ctx context.Context, id string, clusterCertExpiryStatus *storage.ClusterCertExpiryStatus) error
	UpdateClusterHealth(ctx context.Context, id string, clusterHealthStatus *storage.ClusterHealthStatus) error
	UpdateSensorDeploymentIdentification(ctx context.Context, id string, identification *storage.SensorDeploymentIdentification) error
	// UpdateAuditLogFileStates updates for each node in the cluster where the audit log was last at
	// states is a map of node name to the state for that node
	UpdateAuditLogFileStates(ctx context.Context, id string, states map[string]*storage.AuditLogFileState) error

	Search(ctx context.Context, q *v1.Query) ([]pkgSearch.Result, error)
	Count(ctx context.Context, q *v1.Query) (int, error)
	SearchRawClusters(ctx context.Context, q *v1.Query) ([]*storage.Cluster, error)
	SearchResults(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error)

	LookupOrCreateClusterFromConfig(ctx context.Context, clusterID, bundleID string, hello *central.SensorHello) (*storage.Cluster, error)
}

// New returns an instance of DataStore.
func New(
	clusterStorage clusterStore.Store,
	clusterHealthStorage clusterHealthStore.Store,
	clusterCVEs clusterCVEDS.DataStore,
	ads alertDataStore.DataStore,
	imageIntegrationStore imageIntegrationDataStore.DataStore,
	namespaceDS namespaceDataStore.DataStore,
	dds deploymentDataStore.DataStore,
	ns nodeDataStore.GlobalDataStore,
	pods podDataStore.DataStore,
	ss secretDataStore.DataStore,
	flows netFlowsDataStore.ClusterDataStore,
	netEntities netEntityDataStore.EntityDataStore,
	sads serviceAccountDataStore.DataStore,
	rds roleDataStore.DataStore,
	rbds roleBindingDataStore.DataStore,
	cm connection.Manager,
	notifier notifierProcessor.Processor,
	graphProvider graph.Provider,
	clusterRanker *ranking.Ranker,
	indexer index.Indexer,
	networkBaselineMgr networkBaselineManager.Manager,
) (DataStore, error) {
	ds := &datastoreImpl{
		clusterStorage:            clusterStorage,
		clusterHealthStorage:      clusterHealthStorage,
		clusterCVEDataStore:       clusterCVEs,
		alertDataStore:            ads,
		imageIntegrationDataStore: imageIntegrationStore,
		namespaceDataStore:        namespaceDS,
		deploymentDataStore:       dds,
		nodeDataStore:             ns,
		podDataStore:              pods,
		secretsDataStore:          ss,
		netFlowsDataStore:         flows,
		netEntityDataStore:        netEntities,
		serviceAccountDataStore:   sads,
		roleDataStore:             rds,
		roleBindingDataStore:      rbds,
		cm:                        cm,
		notifier:                  notifier,
		clusterRanker:             clusterRanker,
		indexer:                   indexer,
		networkBaselineMgr:        networkBaselineMgr,
		idToNameCache:             simplecache.New(),
		nameToIDCache:             simplecache.New(),
	}

	if env.PostgresDatastoreEnabled.BooleanSetting() {
		ds.searcher = search.NewV2(clusterStorage, indexer, clusterRanker)
	} else {
		ds.searcher = search.New(clusterStorage, indexer, graphProvider, clusterRanker)
	}

	if err := ds.buildIndex(sac.WithAllAccess(context.Background())); err != nil {
		return ds, err
	}

	if err := ds.registerClusterForNetworkGraphExtSrcs(); err != nil {
		return ds, err
	}

	go ds.cleanUpNodeStore(cleanupCtx)
	return ds, nil
}

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
	nodeInternalStore, err := nodeNodeDataStore.GetTestPostgresDataStore(t, pool)
	if err != nil {
		return nil, err
	}
	nodeStore, err := nodeGlobalDataStore.New(nodeInternalStore)
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
	nodeInternalStore, err := nodeNodeDataStore.GetTestRocksBleveDataStore(t, rocksengine, bleveIndex, dacky, keyFence)
	if err != nil {
		return nil, err
	}
	nodeStore, err := nodeGlobalDataStore.New(nodeInternalStore)
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
