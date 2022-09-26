package datastore

import (
	"context"
	"testing"

	"github.com/blevesearch/bleve"
	"github.com/jackc/pgx/v4/pgxpool"
	clusterIndex "github.com/stackrox/rox/central/cluster/index"
	"github.com/stackrox/rox/central/componentcveedge/datastore/store/postgres"
	"github.com/stackrox/rox/central/componentcveedge/index"
	"github.com/stackrox/rox/central/componentcveedge/search"
	"github.com/stackrox/rox/central/componentcveedge/store"
	dackboxStore "github.com/stackrox/rox/central/componentcveedge/store/dackbox"
	cveIndexer "github.com/stackrox/rox/central/cve/index"
	deploymentIndex "github.com/stackrox/rox/central/deployment/index"
	imageIndex "github.com/stackrox/rox/central/image/index"
	componentIndex "github.com/stackrox/rox/central/imagecomponent/index"
	imageComponentEdgeIndex "github.com/stackrox/rox/central/imagecomponentedge/index"
	imageCVEEdgeIndex "github.com/stackrox/rox/central/imagecveedge/index"
	nodeIndex "github.com/stackrox/rox/central/node/index"
	nodeComponentEdgeIndex "github.com/stackrox/rox/central/nodecomponentedge/index"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/dackbox"
	"github.com/stackrox/rox/pkg/dackbox/graph"
	rocksdbBase "github.com/stackrox/rox/pkg/rocksdb"
	searchPkg "github.com/stackrox/rox/pkg/search"
)

// DataStore is an intermediary to Component/CVE edge storage.
//go:generate mockgen-wrapper
type DataStore interface {
	Search(ctx context.Context, q *v1.Query) ([]searchPkg.Result, error)
	SearchEdges(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error)
	SearchRawEdges(ctx context.Context, q *v1.Query) ([]*storage.ComponentCVEEdge, error)

	Exists(ctx context.Context, id string) (bool, error)
	Get(ctx context.Context, id string) (*storage.ComponentCVEEdge, bool, error)
	Count(ctx context.Context, q *v1.Query) (int, error)
}

// New returns a new instance of a DataStore.
func New(graphProvider graph.Provider, storage store.Store, indexer index.Indexer, searcher search.Searcher) DataStore {
	ds := &datastoreImpl{
		storage:       storage,
		indexer:       indexer,
		searcher:      searcher,
		graphProvider: graphProvider,
	}
	return ds
}

// GetTestPostgresDataStore provides a datastore connected to postgres for testing purposes.
func GetTestPostgresDataStore(_ *testing.T, pool *pgxpool.Pool) (DataStore, error) {
	dbstore := postgres.New(pool)
	indexer := postgres.NewIndexer(pool)
	searcher := search.NewV2(dbstore, indexer)
	return New(nil, dbstore, indexer, searcher), nil
}

// GetTestRocksBleveDataStore provides a datastore connected to rocksdb and bleve for testing purposes.
func GetTestRocksBleveDataStore(_ *testing.T, _ *rocksdbBase.RocksDB, bleveIndex bleve.Index, dacky *dackbox.DackBox) (DataStore, error) {
	dbstore, err := dackboxStore.New(dacky)
	if err != nil {
		return nil, err
	}
	indexer := index.New(bleveIndex)
	imageCVEIndexer := cveIndexer.New(bleveIndex)
	componentIndexer := componentIndex.New(bleveIndex)
	imageComponentEdgeIndexer := imageComponentEdgeIndex.New(bleveIndex)
	imageCVEEdgeIndexer := imageCVEEdgeIndex.New(bleveIndex)
	imageIndexer := imageIndex.New(bleveIndex)
	nodeComponentEdgeIndexer := nodeComponentEdgeIndex.New(bleveIndex)
	nodeIndexer := nodeIndex.New(bleveIndex)
	deploymentIndexer := deploymentIndex.New(bleveIndex, bleveIndex)
	clusterIndexer := clusterIndex.New(bleveIndex)
	searcher := search.New(dbstore, dacky, indexer,
		imageCVEIndexer, componentIndexer, imageComponentEdgeIndexer,
		imageCVEEdgeIndexer, imageIndexer, nodeComponentEdgeIndexer, nodeIndexer, deploymentIndexer, clusterIndexer)
	return New(dacky, dbstore, indexer, searcher), nil
}
