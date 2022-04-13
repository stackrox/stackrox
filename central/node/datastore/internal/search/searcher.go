package search

import (
	"context"

	componentCVEEdgeIndexer "github.com/stackrox/stackrox/central/componentcveedge/index"
	cveIndexer "github.com/stackrox/stackrox/central/cve/index"
	componentIndexer "github.com/stackrox/stackrox/central/imagecomponent/index"
	"github.com/stackrox/stackrox/central/node/datastore/internal/store"
	nodeIndexer "github.com/stackrox/stackrox/central/node/index"
	nodeComponentEdgeIndexer "github.com/stackrox/stackrox/central/nodecomponentedge/index"
	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/dackbox/graph"
	"github.com/stackrox/stackrox/pkg/search"
)

// Searcher provides search functionality on existing nodes
//go:generate mockgen-wrapper
type Searcher interface {
	SearchNodes(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error)
	SearchRawNodes(ctx context.Context, q *v1.Query) ([]*storage.Node, error)

	Search(ctx context.Context, q *v1.Query) ([]search.Result, error)
	Count(ctx context.Context, q *v1.Query) (int, error)
}

// New returns a new instance of Searcher for the given storage and indexers.
func New(storage store.Store, graphProvider graph.Provider,
	cveIndexer cveIndexer.Indexer,
	componentCVEEdgeIndexer componentCVEEdgeIndexer.Indexer,
	componentIndexer componentIndexer.Indexer,
	nodeComponentEdgeIndexer nodeComponentEdgeIndexer.Indexer,
	nodeIndexer nodeIndexer.Indexer) Searcher {
	return &searcherImpl{
		storage:       storage,
		indexer:       nodeIndexer,
		graphProvider: graphProvider,
		searcher: formatSearcher(
			cveIndexer,
			componentCVEEdgeIndexer,
			componentIndexer,
			nodeComponentEdgeIndexer,
			nodeIndexer,
		),
	}
}
