package datastore

import (
	"context"
	"testing"

	"github.com/blevesearch/bleve"
	"github.com/jackc/pgx/v4/pgxpool"
	clusterIndex "github.com/stackrox/rox/central/cluster/index"
	componentCVEEdgeIndex "github.com/stackrox/rox/central/componentcveedge/index"
	cveIndex "github.com/stackrox/rox/central/cve/index"
	deploymentIndex "github.com/stackrox/rox/central/deployment/index"
	imageIndex "github.com/stackrox/rox/central/image/index"
	"github.com/stackrox/rox/central/imagecomponent/datastore/store/postgres"
	"github.com/stackrox/rox/central/imagecomponent/index"
	"github.com/stackrox/rox/central/imagecomponent/search"
	"github.com/stackrox/rox/central/imagecomponent/store"
	dackboxStore "github.com/stackrox/rox/central/imagecomponent/store/dackbox"
	imageComponentEdgeIndex "github.com/stackrox/rox/central/imagecomponentedge/index"
	imageCVEEdgeIndex "github.com/stackrox/rox/central/imagecveedge/index"
	nodeIndex "github.com/stackrox/rox/central/node/index"
	nodeComponentEdgeIndex "github.com/stackrox/rox/central/nodecomponentedge/index"
	"github.com/stackrox/rox/central/ranking"
	riskDataStore "github.com/stackrox/rox/central/risk/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/dackbox"
	"github.com/stackrox/rox/pkg/dackbox/concurrency"
	"github.com/stackrox/rox/pkg/dackbox/graph"
	rocksdbBase "github.com/stackrox/rox/pkg/rocksdb"
	searchPkg "github.com/stackrox/rox/pkg/search"
)

// DataStore is an intermediary to ImageComponent storage.
//go:generate mockgen-wrapper
type DataStore interface {
	Search(ctx context.Context, q *v1.Query) ([]searchPkg.Result, error)
	SearchImageComponents(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error)
	SearchRawImageComponents(ctx context.Context, q *v1.Query) ([]*storage.ImageComponent, error)

	Exists(ctx context.Context, id string) (bool, error)
	Get(ctx context.Context, id string) (*storage.ImageComponent, bool, error)
	Count(ctx context.Context, q *v1.Query) (int, error)
	GetBatch(ctx context.Context, id []string) ([]*storage.ImageComponent, error)
}

// New returns a new instance of a DataStore.
func New(graphProvider graph.Provider, storage store.Store, indexer index.Indexer, searcher search.Searcher, risks riskDataStore.DataStore, ranker *ranking.Ranker) DataStore {
	ds := &datastoreImpl{
		storage:              storage,
		indexer:              indexer,
		searcher:             searcher,
		graphProvider:        graphProvider,
		risks:                risks,
		imageComponentRanker: ranker,
	}

	ds.initializeRankers()
	return ds
}

// GetTestPostgresDataStore provides a datastore connected to postgres for testing purposes.
func GetTestPostgresDataStore(t *testing.T, pool *pgxpool.Pool) (DataStore, error) {
	dbstore := postgres.New(pool)
	indexer := postgres.NewIndexer(pool)
	searcher := search.NewV2(dbstore, indexer)
	riskStore, err := riskDataStore.GetTestPostgresDataStore(t, pool)
	if err != nil {
		return nil, err
	}
	return New(nil, dbstore, indexer, searcher, riskStore, ranking.ComponentRanker()), nil
}

// GetTestRocksBleveDataStore provides a datastore connected to rocksdb and bleve for testing purposes.
func GetTestRocksBleveDataStore(t *testing.T, rocksengine *rocksdbBase.RocksDB, bleveIndex bleve.Index, dacky *dackbox.DackBox, keyFence concurrency.KeyFence) (DataStore, error) {
	dbstore, err := dackboxStore.New(dacky, keyFence)
	if err != nil {
		return nil, err
	}
	indexer := index.New(bleveIndex)
	cveIndexer := cveIndex.New(bleveIndex)
	componentCVEEdgeIndexer := componentCVEEdgeIndex.New(bleveIndex)
	imageComponentEdgeIndexer := imageComponentEdgeIndex.New(bleveIndex)
	imageCVEEdgeIndexer := imageCVEEdgeIndex.New(bleveIndex)
	imageIndexer := imageIndex.New(bleveIndex)
	nodeComponentEdgeIndexer := nodeComponentEdgeIndex.New(bleveIndex)
	nodeIndexer := nodeIndex.New(bleveIndex)
	deploymentIndexer := deploymentIndex.New(bleveIndex, bleveIndex)
	clusterIndexer := clusterIndex.New(bleveIndex)
	searcher := search.New(dbstore, dacky, cveIndexer, componentCVEEdgeIndexer, indexer, imageComponentEdgeIndexer,
		imageCVEEdgeIndexer, imageIndexer, nodeComponentEdgeIndexer, nodeIndexer, deploymentIndexer, clusterIndexer)
	riskStore, err := riskDataStore.GetTestRocksBleveDataStore(t, rocksengine, bleveIndex)
	if err != nil {
		return nil, err
	}
	return New(dacky, dbstore, indexer, searcher, riskStore, ranking.ComponentRanker()), nil
}
