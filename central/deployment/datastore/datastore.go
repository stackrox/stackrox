package datastore

import (
	"context"
	"errors"
	"testing"

	"github.com/stackrox/rox/central/deployment/datastore/internal/search"
	"github.com/stackrox/rox/central/deployment/store"
	"github.com/stackrox/rox/central/deployment/store/cache"
	pgStore "github.com/stackrox/rox/central/deployment/store/postgres"
	imageDS "github.com/stackrox/rox/central/image/datastore"
	nfDS "github.com/stackrox/rox/central/networkgraph/flow/datastore"
	pbDS "github.com/stackrox/rox/central/processbaseline/datastore"
	processIndicatorFilter "github.com/stackrox/rox/central/processindicator/filter"
	"github.com/stackrox/rox/central/ranking"
	riskDS "github.com/stackrox/rox/central/risk/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/expiringcache"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/process/filter"
	pkgSearch "github.com/stackrox/rox/pkg/search"
)

// DataStore is an intermediary to DeploymentStorage.
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

func newDataStore(storage store.Store, pool postgres.DB, images imageDS.DataStore, baselines pbDS.DataStore, networkFlows nfDS.ClusterDataStore, risks riskDS.DataStore, deletedDeploymentCache expiringcache.Cache, processFilter filter.Filter, clusterRanker *ranking.Ranker, nsRanker *ranking.Ranker, deploymentRanker *ranking.Ranker) (DataStore, error) {
	storage, err := cache.NewCachedStore(storage)
	if err != nil {
		return nil, err
	}

	searcher := search.NewV2(storage, pgStore.NewIndexer(pool))
	ds := newDatastoreImpl(storage, searcher, images, baselines, networkFlows, risks, deletedDeploymentCache, processFilter, clusterRanker, nsRanker, deploymentRanker)

	ds.initializeRanker()
	return ds, nil
}

// New creates a deployment datastore using postgres.
func New(pool postgres.DB, images imageDS.DataStore, baselines pbDS.DataStore, networkFlows nfDS.ClusterDataStore, risks riskDS.DataStore, deletedDeploymentCache expiringcache.Cache, processFilter filter.Filter, clusterRanker *ranking.Ranker, nsRanker *ranking.Ranker, deploymentRanker *ranking.Ranker) (DataStore, error) {
	return newDataStore(pgStore.NewFullStore(pool), pool, images, baselines, networkFlows, risks, deletedDeploymentCache, processFilter, clusterRanker, nsRanker, deploymentRanker)
}

// NewTestDataStore allows for direct creation of the datastore for testing purposes
func NewTestDataStore(t testing.TB, storage store.Store, pool postgres.DB, images imageDS.DataStore, baselines pbDS.DataStore, networkFlows nfDS.ClusterDataStore, risks riskDS.DataStore, deletedDeploymentCache expiringcache.Cache, processFilter filter.Filter, clusterRanker *ranking.Ranker, nsRanker *ranking.Ranker, deploymentRanker *ranking.Ranker) (DataStore, error) {
	if t == nil {
		return nil, errors.New("NewTestDataStore called without testing")
	}
	storage, err := cache.NewCachedStore(storage)
	if err != nil {
		return nil, err
	}

	searcher := search.NewV2(storage, pgStore.NewIndexer(pool))
	ds := newDatastoreImpl(storage, searcher, images, baselines, networkFlows, risks, deletedDeploymentCache, processFilter, clusterRanker, nsRanker, deploymentRanker)

	ds.initializeRanker()
	return ds, nil
}

// GetTestPostgresDataStore provides a datastore connected to postgres for testing purposes.
func GetTestPostgresDataStore(t testing.TB, pool postgres.DB) (DataStore, error) {
	dbstore := pgStore.FullStoreWrap(pgStore.New(pool))
	indexer := pgStore.NewIndexer(pool)
	searcher := search.NewV2(dbstore, indexer)
	imageStore := imageDS.GetTestPostgresDataStore(t, pool)
	processBaselineStore, err := pbDS.GetTestPostgresDataStore(t, pool)
	if err != nil {
		return nil, err
	}
	networkFlowClusterStore, err := nfDS.GetTestPostgresClusterDataStore(t, pool)
	if err != nil {
		return nil, err
	}
	riskStore := riskDS.GetTestPostgresDataStore(t, pool)
	processFilter := processIndicatorFilter.Singleton()
	clusterRanker := ranking.ClusterRanker()
	namespaceRanker := ranking.NamespaceRanker()
	deploymentRanker := ranking.DeploymentRanker()
	return newDatastoreImpl(dbstore, searcher, imageStore, processBaselineStore, networkFlowClusterStore, riskStore, nil, processFilter, clusterRanker, namespaceRanker, deploymentRanker), nil
}
