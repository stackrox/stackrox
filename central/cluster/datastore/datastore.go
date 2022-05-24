package datastore

import (
	"context"

	alertDataStore "github.com/stackrox/rox/central/alert/datastore"
	"github.com/stackrox/rox/central/cluster/datastore/internal/search"
	"github.com/stackrox/rox/central/cluster/index"
	clusterStore "github.com/stackrox/rox/central/cluster/store/cluster"
	clusterHealthStore "github.com/stackrox/rox/central/cluster/store/clusterhealth"
	deploymentDataStore "github.com/stackrox/rox/central/deployment/datastore"
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
	"github.com/stackrox/rox/central/role/resources"
	secretDataStore "github.com/stackrox/rox/central/secret/datastore"
	"github.com/stackrox/rox/central/sensor/service/connection"
	serviceAccountDataStore "github.com/stackrox/rox/central/serviceaccount/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/dackbox/graph"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sac"
	pkgSearch "github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/simplecache"
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
	indexer index.Indexer,
	ads alertDataStore.DataStore,
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
	networkBaselineMgr networkBaselineManager.Manager,
) (DataStore, error) {
	ds := &datastoreImpl{
		clusterStorage:          clusterStorage,
		clusterHealthStorage:    clusterHealthStorage,
		indexer:                 indexer,
		alertDataStore:          ads,
		namespaceDataStore:      namespaceDS,
		deploymentDataStore:     dds,
		nodeDataStore:           ns,
		podDataStore:            pods,
		secretsDataStore:        ss,
		netFlowsDataStore:       flows,
		netEntityDataStore:      netEntities,
		serviceAccountDataStore: sads,
		roleDataStore:           rds,
		roleBindingDataStore:    rbds,
		cm:                      cm,
		notifier:                notifier,
		clusterRanker:           clusterRanker,
		networkBaselineMgr:      networkBaselineMgr,

		idToNameCache: simplecache.New(),
		nameToIDCache: simplecache.New(),
	}

	if features.PostgresDatastore.Enabled() {
		ds.searcher = search.NewV2(clusterStorage, indexer, clusterRanker)
	} else {
		ds.searcher = search.New(clusterStorage, indexer, graphProvider, clusterRanker)
	}
	ctx := sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.Cluster)))
	if err := ds.buildIndex(ctx); err != nil {
		return ds, err
	}

	if err := ds.registerClusterForNetworkGraphExtSrcs(); err != nil {
		return ds, err
	}

	go ds.cleanUpNodeStore(cleanupCtx)
	return ds, nil
}
