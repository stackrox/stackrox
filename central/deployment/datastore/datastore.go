package datastore

import (
	"context"

	"github.com/blevesearch/bleve"
	"github.com/stackrox/stackrox/central/analystnotes"
	componentCVEEdgeIndexer "github.com/stackrox/stackrox/central/componentcveedge/index"
	cveIndexer "github.com/stackrox/stackrox/central/cve/index"
	"github.com/stackrox/stackrox/central/deployment/datastore/internal/processtagsstore"
	"github.com/stackrox/stackrox/central/deployment/datastore/internal/search"
	"github.com/stackrox/stackrox/central/deployment/index"
	"github.com/stackrox/stackrox/central/deployment/store"
	"github.com/stackrox/stackrox/central/deployment/store/cache"
	dackBoxStore "github.com/stackrox/stackrox/central/deployment/store/dackbox"
	"github.com/stackrox/stackrox/central/deployment/store/postgres"
	"github.com/stackrox/stackrox/central/globaldb"
	imageDS "github.com/stackrox/stackrox/central/image/datastore"
	imageIndexer "github.com/stackrox/stackrox/central/image/index"
	componentIndexer "github.com/stackrox/stackrox/central/imagecomponent/index"
	imageComponentEdgeIndexer "github.com/stackrox/stackrox/central/imagecomponentedge/index"
	imageCVEEdgeIndexer "github.com/stackrox/stackrox/central/imagecveedge/index"
	nfDS "github.com/stackrox/stackrox/central/networkgraph/flow/datastore"
	pbDS "github.com/stackrox/stackrox/central/processbaseline/datastore"
	"github.com/stackrox/stackrox/central/ranking"
	riskDS "github.com/stackrox/stackrox/central/risk/datastore"
	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/concurrency"
	"github.com/stackrox/stackrox/pkg/dackbox"
	"github.com/stackrox/stackrox/pkg/dackbox/graph"
	"github.com/stackrox/stackrox/pkg/expiringcache"
	"github.com/stackrox/stackrox/pkg/features"
	"github.com/stackrox/stackrox/pkg/process/filter"
	pkgSearch "github.com/stackrox/stackrox/pkg/search"
)

// DataStore is an intermediary to AlertStorage.
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

	AddTagsToProcessKey(ctx context.Context, key *analystnotes.ProcessNoteKey, tags []string) error
	RemoveTagsFromProcessKey(ctx context.Context, key *analystnotes.ProcessNoteKey, tags []string) error
	GetTagsForProcessKey(ctx context.Context, key *analystnotes.ProcessNoteKey) ([]string, error)

	RemoveDeployment(ctx context.Context, clusterID, id string) error

	GetImagesForDeployment(ctx context.Context, deployment *storage.Deployment) ([]*storage.Image, error)
	GetDeploymentIDs(ctx context.Context) ([]string, error)
}

func newDataStore(storage store.Store, graphProvider graph.Provider, processTagsStore processtagsstore.Store, bleveIndex bleve.Index, processIndex bleve.Index,
	images imageDS.DataStore, baselines pbDS.DataStore, networkFlows nfDS.ClusterDataStore,
	risks riskDS.DataStore, deletedDeploymentCache expiringcache.Cache, processFilter filter.Filter,
	clusterRanker *ranking.Ranker, nsRanker *ranking.Ranker, deploymentRanker *ranking.Ranker) DataStore {
	storage = cache.NewCachedStore(storage)
	var deploymentIndexer index.Indexer
	var searcher search.Searcher
	if features.PostgresDatastore.Enabled() {
		deploymentIndexer = postgres.NewIndexer(globaldb.GetPostgres())
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
	ds := newDatastoreImpl(storage, processTagsStore, deploymentIndexer, searcher, images, baselines, networkFlows, risks,
		deletedDeploymentCache, processFilter, clusterRanker, nsRanker, deploymentRanker)

	ds.initializeRanker()
	return ds
}

// New creates a deployment datastore based on dackbox
func New(dacky *dackbox.DackBox, keyFence concurrency.KeyFence, processTagsStore processtagsstore.Store, bleveIndex bleve.Index, processIndex bleve.Index,
	images imageDS.DataStore, baselines pbDS.DataStore, networkFlows nfDS.ClusterDataStore,
	risks riskDS.DataStore, deletedDeploymentCache expiringcache.Cache, processFilter filter.Filter,
	clusterRanker *ranking.Ranker, nsRanker *ranking.Ranker, deploymentRanker *ranking.Ranker) DataStore {
	var storage store.Store
	if features.PostgresDatastore.Enabled() {
		storage = postgres.NewFullStore(context.TODO(), globaldb.GetPostgres())
	} else {
		storage = dackBoxStore.New(dacky, keyFence)
	}
	return newDataStore(storage, dacky, processTagsStore, bleveIndex, processIndex, images, baselines, networkFlows, risks, deletedDeploymentCache, processFilter, clusterRanker, nsRanker, deploymentRanker)
}
