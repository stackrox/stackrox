package datastore

import (
	"context"
	"testing"

	"github.com/blevesearch/bleve"
	"github.com/gogo/protobuf/types"
	clusterIndex "github.com/stackrox/rox/central/cluster/index"
	clusterCVEEdgeIndex "github.com/stackrox/rox/central/clustercveedge/index"
	componentCVEEdgeIndex "github.com/stackrox/rox/central/componentcveedge/index"
	"github.com/stackrox/rox/central/cve/common"
	"github.com/stackrox/rox/central/cve/index"
	"github.com/stackrox/rox/central/cve/search"
	"github.com/stackrox/rox/central/cve/store"
	dackboxStore "github.com/stackrox/rox/central/cve/store/dackbox"
	deploymentIndex "github.com/stackrox/rox/central/deployment/index"
	imageIndex "github.com/stackrox/rox/central/image/index"
	componentIndex "github.com/stackrox/rox/central/imagecomponent/index"
	imageComponentEdgeIndex "github.com/stackrox/rox/central/imagecomponentedge/index"
	imageCVEEdgeIndex "github.com/stackrox/rox/central/imagecveedge/index"
	nodeIndex "github.com/stackrox/rox/central/node/index"
	nodeComponentEdgeIndex "github.com/stackrox/rox/central/nodecomponentedge/index"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/dackbox"
	"github.com/stackrox/rox/pkg/dackbox/graph"
	"github.com/stackrox/rox/pkg/dackbox/utils/queue"
	rocksdbBase "github.com/stackrox/rox/pkg/rocksdb"
	searchPkg "github.com/stackrox/rox/pkg/search"
)

// DataStore is an intermediary to CVE storage.
//go:generate mockgen-wrapper
type DataStore interface {
	Search(ctx context.Context, q *v1.Query) ([]searchPkg.Result, error)
	SearchCVEs(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error)
	SearchRawCVEs(ctx context.Context, q *v1.Query) ([]*storage.CVE, error)

	Exists(ctx context.Context, id string) (bool, error)
	Get(ctx context.Context, id string) (*storage.CVE, bool, error)
	Count(ctx context.Context, q *v1.Query) (int, error)
	GetBatch(ctx context.Context, id []string) ([]*storage.CVE, error)

	Suppress(ctx context.Context, start *types.Timestamp, duration *types.Duration, ids ...string) error
	Unsuppress(ctx context.Context, ids ...string) error
	EnrichImageWithSuppressedCVEs(image *storage.Image)
	EnrichNodeWithSuppressedCVEs(node *storage.Node)

	Delete(ctx context.Context, ids ...string) error
}

// New returns a new instance of a DataStore.
func New(graphProvider graph.Provider, indexQ queue.WaitableQueue, storage store.Store, indexer index.Indexer, searcher search.Searcher) (DataStore, error) {
	ds := &datastoreImpl{
		storage:       storage,
		indexer:       indexer,
		searcher:      searcher,
		graphProvider: graphProvider,
		indexQ:        indexQ,

		cveSuppressionCache: make(common.CVESuppressionCache),
	}
	if err := ds.buildSuppressedCache(); err != nil {
		return nil, err
	}
	return ds, nil
}

// GetTestRocksBleveDataStore provides a datastore connected to rocksdb and bleve for testing purposes.
func GetTestRocksBleveDataStore(_ *testing.T, _ *rocksdbBase.RocksDB, bleveIndex bleve.Index, dacky *dackbox.DackBox, keyFence concurrency.KeyFence, indexQ queue.WaitableQueue) (DataStore, error) {
	dbstore := dackboxStore.New(dacky, keyFence)
	indexer := index.New(bleveIndex)
	clusterCVEEdgeIndexer := clusterCVEEdgeIndex.New(bleveIndex)
	componentCVEEdgeIndexer := componentCVEEdgeIndex.New(bleveIndex)
	componentIndexer := componentIndex.New(bleveIndex)
	imageComponentEdgeIndexer := imageComponentEdgeIndex.New(bleveIndex)
	imageCVEEdgeIndexer := imageCVEEdgeIndex.New(bleveIndex)
	imageIndexer := imageIndex.New(bleveIndex)
	nodeComponentEdgeIndexer := nodeComponentEdgeIndex.New(bleveIndex)
	nodeIndexer := nodeIndex.New(bleveIndex)
	deploymentIndexer := deploymentIndex.New(bleveIndex, bleveIndex)
	clusterIndexer := clusterIndex.New(bleveIndex)
	searcher := search.New(dbstore, dacky, indexer,
		clusterCVEEdgeIndexer, componentCVEEdgeIndexer, componentIndexer, imageComponentEdgeIndexer,
		imageCVEEdgeIndexer, imageIndexer, nodeComponentEdgeIndexer, nodeIndexer, deploymentIndexer, clusterIndexer)
	return New(dacky, indexQ, dbstore, indexer, searcher)
}
