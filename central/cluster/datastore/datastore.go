package datastore

import (
	"context"
	"time"

	alertDataStore "github.com/stackrox/rox/central/alert/datastore"
	"github.com/stackrox/rox/central/cluster/datastore/internal/search"
	"github.com/stackrox/rox/central/cluster/index"
	"github.com/stackrox/rox/central/cluster/store"
	deploymentDataStore "github.com/stackrox/rox/central/deployment/datastore"
	nodeDataStore "github.com/stackrox/rox/central/node/globaldatastore"
	notifierProcessor "github.com/stackrox/rox/central/notifier/processor"
	"github.com/stackrox/rox/central/ranking"
	"github.com/stackrox/rox/central/role/resources"
	secretDataStore "github.com/stackrox/rox/central/secret/datastore"
	"github.com/stackrox/rox/central/sensor/service/connection"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/dackbox/graph"
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
//go:generate mockgen-wrapper DataStore
type DataStore interface {
	GetCluster(ctx context.Context, id string) (*storage.Cluster, bool, error)
	GetClusterName(ctx context.Context, id string) (string, bool, error)
	GetClusters(ctx context.Context) ([]*storage.Cluster, error)
	CountClusters(ctx context.Context) (int, error)
	Exists(ctx context.Context, id string) (bool, error)

	AddCluster(ctx context.Context, cluster *storage.Cluster) (string, error)
	UpdateCluster(ctx context.Context, cluster *storage.Cluster) error
	RemoveCluster(ctx context.Context, id string, done *concurrency.Signal) error
	UpdateClusterContactTimes(ctx context.Context, t time.Time, ids ...string) error
	// UpdateClusterStatus updates the cluster status. Note that any cluster-upgrade status
	// in this endpoint will be ignored. To insert that, callers MUST separately call
	// UpdateClusterUpgradeStatus.
	UpdateClusterStatus(ctx context.Context, id string, status *storage.ClusterStatus) error
	UpdateClusterUpgradeStatus(ctx context.Context, id string, clusterUpgradeStatus *storage.ClusterUpgradeStatus) error
	Search(ctx context.Context, q *v1.Query) ([]pkgSearch.Result, error)
	SearchRawClusters(ctx context.Context, q *v1.Query) ([]*storage.Cluster, error)
	SearchResults(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error)
}

// New returns an instance of DataStore.
func New(
	storage store.Store,
	indexer index.Indexer,
	ads alertDataStore.DataStore,
	dds deploymentDataStore.DataStore,
	ns nodeDataStore.GlobalDataStore,
	ss secretDataStore.DataStore,
	cm connection.Manager,
	notifier notifierProcessor.Processor,
	graphProvider graph.Provider,
	clusterRanker *ranking.Ranker) (DataStore, error) {
	ds := &datastoreImpl{
		storage:       storage,
		indexer:       indexer,
		searcher:      search.New(storage, indexer, graphProvider, clusterRanker),
		ads:           ads,
		dds:           dds,
		ns:            ns,
		ss:            ss,
		cm:            cm,
		notifier:      notifier,
		clusterRanker: clusterRanker,

		cache: simplecache.New(),
	}
	if err := ds.buildIndex(); err != nil {
		return ds, err
	}

	go ds.cleanUpNodeStore(cleanupCtx)
	return ds, nil
}
