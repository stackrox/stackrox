package datastore

import (
	"context"

	alertDataStore "github.com/stackrox/rox/central/alert/datastore"
	"github.com/stackrox/rox/central/cluster/datastore/internal/search"
	"github.com/stackrox/rox/central/cluster/index"
	clusterStore "github.com/stackrox/rox/central/cluster/store/cluster"
	clusterHealthStore "github.com/stackrox/rox/central/cluster/store/clusterhealth"
	clusterCVEDS "github.com/stackrox/rox/central/cve/cluster/datastore"
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
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/cache/objectarraycache"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/logging"
	notifierProcessor "github.com/stackrox/rox/pkg/notifier"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/effectiveaccessscope"
	pkgSearch "github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/simplecache"
)

var (
	log = logging.LoggerForModule()
)

// DataStore is the entry point for modifying Cluster data.
//
//go:generate mockgen-wrapper
type DataStore interface {
	GetCluster(ctx context.Context, id string) (*storage.Cluster, bool, error)
	GetClusterName(ctx context.Context, id string) (string, bool, error)
	GetClusters(ctx context.Context) ([]*storage.Cluster, error)
	GetClustersForSAC(ctx context.Context) ([]effectiveaccessscope.ClusterForSAC, error)
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
	ns nodeDataStore.DataStore,
	pods podDataStore.DataStore,
	ss secretDataStore.DataStore,
	flows netFlowsDataStore.ClusterDataStore,
	netEntities netEntityDataStore.EntityDataStore,
	sads serviceAccountDataStore.DataStore,
	rds roleDataStore.DataStore,
	rbds roleBindingDataStore.DataStore,
	cm connection.Manager,
	notifier notifierProcessor.Processor,
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
		networkBaselineMgr:        networkBaselineMgr,
		idToNameCache:             simplecache.New(),
		nameToIDCache:             simplecache.New(),
	}

	ds.objectCacheForSAC = objectarraycache.NewObjectArrayCache(cacheRefreshPeriod, ds.getClustersForSAC)

	ds.searcher = search.NewV2(clusterStorage, indexer, clusterRanker)
	if err := ds.buildCache(sac.WithAllAccess(context.Background())); err != nil {
		return ds, err
	}

	if err := ds.registerClusterForNetworkGraphExtSrcs(); err != nil {
		return ds, err
	}
	return ds, nil
}
