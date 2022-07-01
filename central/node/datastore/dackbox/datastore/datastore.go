package datastore

import (
	"context"
	"testing"

	"github.com/blevesearch/bleve"
	"github.com/jackc/pgx/v4/pgxpool"
	componentCVEEdgeIndexer "github.com/stackrox/rox/central/componentcveedge/index"
	cveIndexer "github.com/stackrox/rox/central/cve/index"
	componentIndexer "github.com/stackrox/rox/central/imagecomponent/index"
	"github.com/stackrox/rox/central/node/datastore/internal/search"
	"github.com/stackrox/rox/central/node/datastore/internal/store"
	dackBoxStore "github.com/stackrox/rox/central/node/datastore/internal/store/dackbox"
	postgresStore "github.com/stackrox/rox/central/node/datastore/internal/store/postgres"
	nodeIndexer "github.com/stackrox/rox/central/node/index"
	nodeComponentEdgeIndexer "github.com/stackrox/rox/central/nodecomponentedge/index"
	"github.com/stackrox/rox/central/ranking"
	riskDS "github.com/stackrox/rox/central/risk/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/dackbox"
	rocksdbBase "github.com/stackrox/rox/pkg/rocksdb"
	searchPkg "github.com/stackrox/rox/pkg/search"
)

// DataStore is an intermediary to NodeStorage.
//go:generate mockgen-wrapper
type DataStore interface {
	Search(ctx context.Context, q *v1.Query) ([]searchPkg.Result, error)
	Count(ctx context.Context, q *v1.Query) (int, error)
	SearchNodes(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error)
	SearchRawNodes(ctx context.Context, q *v1.Query) ([]*storage.Node, error)

	CountNodes(ctx context.Context) (int, error)
	GetNode(ctx context.Context, id string) (*storage.Node, bool, error)
	GetNodesBatch(ctx context.Context, ids []string) ([]*storage.Node, error)

	UpsertNode(ctx context.Context, node *storage.Node) error

	DeleteNodes(ctx context.Context, ids ...string) error
	Exists(ctx context.Context, id string) (bool, error)
}

// newDatastore returns a datastore for Nodes.
// noUpdateTimestamps controls whether timestamps are automatically updated when upserting nodes.
// This should be set to `false` except for some tests.
func newDatastore(dacky *dackbox.DackBox, keyFence concurrency.KeyFence, bleveIndex bleve.Index, noUpdateTimestamps bool, risks riskDS.DataStore, nodeRanker *ranking.Ranker, nodeComponentRanker *ranking.Ranker) DataStore {
	dataStore := dackBoxStore.New(dacky, keyFence, noUpdateTimestamps)
	indexer := nodeIndexer.New(bleveIndex)

	searcher := search.New(dataStore,
		dacky,
		cveIndexer.New(bleveIndex),
		componentCVEEdgeIndexer.New(bleveIndex),
		componentIndexer.New(bleveIndex),
		nodeComponentEdgeIndexer.New(bleveIndex),
		indexer,
	)
	ds := newDatastoreImpl(dataStore, indexer, searcher, risks, nodeRanker, nodeComponentRanker)
	ds.initializeRankers()

	return ds
}

// New returns a new instance of DataStore using the input store, indexer, and searcher.
func New(dacky *dackbox.DackBox, keyFence concurrency.KeyFence, bleveIndex bleve.Index, risks riskDS.DataStore, nodeRanker *ranking.Ranker, nodeComponentRanker *ranking.Ranker) DataStore {
	return newDatastore(dacky, keyFence, bleveIndex, false, risks, nodeRanker, nodeComponentRanker)
}

// NewWithPostgres returns a new instance of DataStore using the input store, indexer, and searcher.
func NewWithPostgres(storage store.Store, indexer nodeIndexer.Indexer, searcher search.Searcher, risks riskDS.DataStore, nodeRanker *ranking.Ranker, nodeComponentRanker *ranking.Ranker) DataStore {
	ds := newDatastoreImpl(storage, indexer, searcher, risks, nodeRanker, nodeComponentRanker)
	ds.initializeRankers()
	return ds
}

// GetTestPostgresDataStore provides a datastore connected to postgres for testing purposes.
func GetTestPostgresDataStore(t *testing.T, pool *pgxpool.Pool) (DataStore, error) {
	dbstore := postgresStore.New(pool, false)
	indexer := postgresStore.NewIndexer(pool)
	searcher := search.NewV2(dbstore, indexer)
	riskStore, err := riskDS.GetTestPostgresDataStore(t, pool)
	if err != nil {
		return nil, err
	}
	nodeRanker := ranking.NodeRanker()
	nodeComponentRanker := ranking.NodeComponentRanker()
	return NewWithPostgres(dbstore, indexer, searcher, riskStore, nodeRanker, nodeComponentRanker), nil
}

// GetTestRocksBleveDataStore provides a datastore connected to rocksdb and bleve for testing purposes.
func GetTestRocksBleveDataStore(t *testing.T, rocksengine *rocksdbBase.RocksDB, bleveIndex bleve.Index, dacky *dackbox.DackBox, keyFence concurrency.KeyFence) (DataStore, error) {
	riskStore, err := riskDS.GetTestRocksBleveDataStore(t, rocksengine, bleveIndex)
	if err != nil {
		return nil, err
	}
	nodeRanker := ranking.NodeRanker()
	nodeComponentRanker := ranking.NodeComponentRanker()
	return New(dacky, keyFence, bleveIndex, riskStore, nodeRanker, nodeComponentRanker), nil
}
