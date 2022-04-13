package datastore

import (
	"context"

	"github.com/blevesearch/bleve"
	componentCVEEdgeIndexer "github.com/stackrox/stackrox/central/componentcveedge/index"
	cveIndexer "github.com/stackrox/stackrox/central/cve/index"
	componentIndexer "github.com/stackrox/stackrox/central/imagecomponent/index"
	"github.com/stackrox/stackrox/central/node/datastore/internal/search"
	dackBoxStore "github.com/stackrox/stackrox/central/node/datastore/internal/store/dackbox"
	nodeIndexer "github.com/stackrox/stackrox/central/node/index"
	nodeComponentEdgeIndexer "github.com/stackrox/stackrox/central/nodecomponentedge/index"
	"github.com/stackrox/stackrox/central/ranking"
	riskDS "github.com/stackrox/stackrox/central/risk/datastore"
	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/concurrency"
	"github.com/stackrox/stackrox/pkg/dackbox"
	searchPkg "github.com/stackrox/stackrox/pkg/search"
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
