package datastore

import (
	"context"
	"errors"
	"testing"

	"github.com/blevesearch/bleve"
	"github.com/jackc/pgx/v4/pgxpool"
	componentCVEEdgeIndexer "github.com/stackrox/rox/central/componentcveedge/index"
	cveIndexer "github.com/stackrox/rox/central/cve/index"
	"github.com/stackrox/rox/central/deployment/datastore/internal/search"
	"github.com/stackrox/rox/central/deployment/index"
	"github.com/stackrox/rox/central/deployment/store"
	"github.com/stackrox/rox/central/deployment/store/cache"
	dackBoxStore "github.com/stackrox/rox/central/deployment/store/dackbox"
	"github.com/stackrox/rox/central/deployment/store/postgres"
	imageDS "github.com/stackrox/rox/central/image/datastore"
	imageIndexer "github.com/stackrox/rox/central/image/index"
	componentIndexer "github.com/stackrox/rox/central/imagecomponent/index"
	imageComponentEdgeIndexer "github.com/stackrox/rox/central/imagecomponentedge/index"
	imageCVEEdgeIndexer "github.com/stackrox/rox/central/imagecveedge/index"
	nfDS "github.com/stackrox/rox/central/networkgraph/flow/datastore"
	pbDS "github.com/stackrox/rox/central/processbaseline/datastore"
	processIndicatorFilter "github.com/stackrox/rox/central/processindicator/filter"
	"github.com/stackrox/rox/central/ranking"
	riskDS "github.com/stackrox/rox/central/risk/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/dackbox"
	"github.com/stackrox/rox/pkg/dackbox/concurrency"
	"github.com/stackrox/rox/pkg/dackbox/graph"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/expiringcache"
	"github.com/stackrox/rox/pkg/process/filter"
	rocksdbBase "github.com/stackrox/rox/pkg/rocksdb"
	pkgSearch "github.com/stackrox/rox/pkg/search"
)

// DataStore is an intermediary to AlertStorage.
//
//go:generate mockgen-wrapper
type DataStore interface {
	Search(ctx context.Context, q *v1.Query) ([]pkgSearch.Result, error)
	Count(ctx context.Context, q *v1.Query) (int, error)
	SearchDeployments(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error)
	SearchRawDeployments(ctx context.Context, q *v1.Query) ([]*storage.Deployment, error)
	SearchListDeployments(ctx context.Context, q *v1.Query) ([]*storage.ListDeployment, error)

	ListDeployment(ctx context.Context, id string) (*storage.ListDeployment, bool, error)

	GetDeployment(ctx context.Context, id string) (*storage.Deployment, bool, error)
	GetDeployments(ctx context.Context, ids []string) ([]*storage.Deployment, error)
	CountDeployments(ctx context.Context) (int, error)
	// UpsertDeployment adds or updates a deployment. If the deployment exists, the tags in the deployment are taken from
	// the stored deployment.
	UpsertDeployment(ctx context.Context, deployment *storage.Deployment) error

	RemoveDeployment(ctx context.Context, clusterID, id string) error

	GetImagesForDeployment(ctx context.Context, deployment *storage.Deployment) ([]*storage.Image, error)
	GetDeploymentIDs(ctx context.Context) ([]string, error)
}

func newDataStore(storage store.Store, graphProvider graph.Provider, pool *pgxpool.Pool,
	bleveIndex bleve.Index, processIndex bleve.Index,
	images imageDS.DataStore, baselines pbDS.DataStore, networkFlows nfDS.ClusterDataStore,
	risks riskDS.DataStore, deletedDeploymentCache expiringcache.Cache, processFilter filter.Filter,
	clusterRanker *ranking.Ranker, nsRanker *ranking.Ranker, deploymentRanker *ranking.Ranker) (DataStore, error) {
	storage, err := cache.NewCachedStore(storage)
	if err != nil {
		return nil, err
	}
	var deploymentIndexer index.Indexer
	var searcher search.Searcher
	if env.PostgresDatastoreEnabled.BooleanSetting() {
		deploymentIndexer = postgres.NewIndexer(pool)
		searcher = search.NewV2(storage, deploymentIndexer)
	} else {
		deploymentIndexer = index.New(bleveIndex, processIndex)
		searcher = search.New(storage,
			graphProvider,
			cveIndexer.New(bleveIndex),
			componentCVEEdgeIndexer.New(bleveIndex),
			componentIndexer.New(bleveIndex),
			imageComponentEdgeIndexer.New(bleveIndex),
			imageIndexer.New(bleveIndex),
			deploymentIndexer,
			imageCVEEdgeIndexer.New(bleveIndex))
	}
	ds := newDatastoreImpl(storage, deploymentIndexer, searcher, images, baselines, networkFlows, risks, deletedDeploymentCache, processFilter, clusterRanker, nsRanker, deploymentRanker)

	ds.initializeRanker()
	return ds, nil
}

// New creates a deployment datastore based on dackbox
func New(dacky *dackbox.DackBox, keyFence concurrency.KeyFence, pool *pgxpool.Pool,
	bleveIndex bleve.Index, processIndex bleve.Index,
	images imageDS.DataStore, baselines pbDS.DataStore, networkFlows nfDS.ClusterDataStore,
	risks riskDS.DataStore, deletedDeploymentCache expiringcache.Cache, processFilter filter.Filter,
	clusterRanker *ranking.Ranker, nsRanker *ranking.Ranker, deploymentRanker *ranking.Ranker) (DataStore, error) {
	var storage store.Store
	if env.PostgresDatastoreEnabled.BooleanSetting() {
		storage = postgres.NewFullStore(pool)
	} else {
		storage = dackBoxStore.New(dacky, keyFence)
	}
	return newDataStore(storage, dacky, pool, bleveIndex, processIndex, images, baselines, networkFlows, risks, deletedDeploymentCache, processFilter, clusterRanker, nsRanker, deploymentRanker)
}

// NewTestDataStore allows for direct creation of the datastore for testing purposes
func NewTestDataStore(t *testing.T, storage store.Store, graphProvider graph.Provider, pool *pgxpool.Pool,
	bleveIndex bleve.Index, processIndex bleve.Index,
	images imageDS.DataStore, baselines pbDS.DataStore, networkFlows nfDS.ClusterDataStore,
	risks riskDS.DataStore, deletedDeploymentCache expiringcache.Cache, processFilter filter.Filter,
	clusterRanker *ranking.Ranker, nsRanker *ranking.Ranker, deploymentRanker *ranking.Ranker) (DataStore, error) {
	if t == nil {
		return nil, errors.New("NewTestDataStore called without testing")
	}
	storage, err := cache.NewCachedStore(storage)
	if err != nil {
		return nil, err
	}
	var deploymentIndexer index.Indexer
	var searcher search.Searcher
	if env.PostgresDatastoreEnabled.BooleanSetting() {
		deploymentIndexer = postgres.NewIndexer(pool)
		searcher = search.NewV2(storage, deploymentIndexer)
	} else {
		deploymentIndexer = index.New(bleveIndex, processIndex)
		searcher = search.New(storage,
			graphProvider,
			cveIndexer.New(bleveIndex),
			componentCVEEdgeIndexer.New(bleveIndex),
			componentIndexer.New(bleveIndex),
			imageComponentEdgeIndexer.New(bleveIndex),
			imageIndexer.New(bleveIndex),
			deploymentIndexer,
			imageCVEEdgeIndexer.New(bleveIndex))
	}
	ds := newDatastoreImpl(storage, deploymentIndexer, searcher, images, baselines, networkFlows, risks, deletedDeploymentCache, processFilter, clusterRanker, nsRanker, deploymentRanker)

	ds.initializeRanker()
	return ds, nil
}

// GetTestPostgresDataStore provides a datastore connected to postgres for testing purposes.
func GetTestPostgresDataStore(t *testing.T, pool *pgxpool.Pool) (DataStore, error) {
	dbstore := postgres.FullStoreWrap(postgres.New(pool))
	indexer := postgres.NewIndexer(pool)
	searcher := search.NewV2(dbstore, indexer)
	imageStore, err := imageDS.GetTestPostgresDataStore(t, pool)
	if err != nil {
		return nil, err
	}
	processBaselineStore, err := pbDS.GetTestPostgresDataStore(t, pool)
	if err != nil {
		return nil, err
	}
	networkFlowClusterStore, err := nfDS.GetTestPostgresClusterDataStore(t, pool)
	if err != nil {
		return nil, err
	}
	riskStore, err := riskDS.GetTestPostgresDataStore(t, pool)
	if err != nil {
		return nil, err
	}
	processFilter := processIndicatorFilter.Singleton()
	clusterRanker := ranking.ClusterRanker()
	namespaceRanker := ranking.NamespaceRanker()
	deploymentRanker := ranking.DeploymentRanker()
	return newDatastoreImpl(dbstore, indexer, searcher, imageStore, processBaselineStore, networkFlowClusterStore, riskStore, nil, processFilter, clusterRanker, namespaceRanker, deploymentRanker), nil
}

// GetTestRocksBleveDataStore provides a datastore connected to rocksdb and bleve for testing purposes.
func GetTestRocksBleveDataStore(t *testing.T, rocksengine *rocksdbBase.RocksDB, bleveIndex bleve.Index, dacky *dackbox.DackBox, keyFence concurrency.KeyFence) (DataStore, error) {
	imageStore, err := imageDS.GetTestRocksBleveDataStore(t, rocksengine, bleveIndex, dacky, keyFence)
	if err != nil {
		return nil, err
	}
	processBaselineStore, err := pbDS.GetTestRocksBleveDataStore(t, rocksengine, bleveIndex)
	if err != nil {
		return nil, err
	}
	networkFlowClusterStore, err := nfDS.GetTestRocksBleveClusterDataStore(t, rocksengine)
	if err != nil {
		return nil, err
	}
	riskStore, err := riskDS.GetTestRocksBleveDataStore(t, rocksengine, bleveIndex)
	if err != nil {
		return nil, err
	}
	processFilter := processIndicatorFilter.Singleton()
	clusterRanker := ranking.ClusterRanker()
	namespaceRanker := ranking.NamespaceRanker()
	deploymentRanker := ranking.DeploymentRanker()
	return New(dacky, keyFence, nil, bleveIndex, bleveIndex, imageStore,
		processBaselineStore, networkFlowClusterStore, riskStore, nil,
		processFilter, clusterRanker, namespaceRanker, deploymentRanker)
}
