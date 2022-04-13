package search

import (
	"context"

	"github.com/stackrox/stackrox/central/clustercveedge/index"
	"github.com/stackrox/stackrox/central/clustercveedge/store"
	cveIndex "github.com/stackrox/stackrox/central/cve/index"
	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/dackbox/graph"
	"github.com/stackrox/stackrox/pkg/search"
)

// Searcher provides search functionality on existing cves.
//go:generate mockgen-wrapper
type Searcher interface {
	Search(ctx context.Context, query *v1.Query) ([]search.Result, error)
	Count(ctx context.Context, query *v1.Query) (int, error)
	SearchEdges(context.Context, *v1.Query) ([]*v1.SearchResult, error)
	SearchRawEdges(ctx context.Context, query *v1.Query) ([]*storage.ClusterCVEEdge, error)
}

// New returns a new instance of Searcher for the given storage and index.
func New(storage store.Store, clusterCVEEdgeIndexer index.Indexer, cveIndexer cveIndex.Indexer, graphProvider graph.Provider) Searcher {
	return &searcherImpl{
		storage:       storage,
		indexer:       clusterCVEEdgeIndexer,
		searcher:      formatSearcher(clusterCVEEdgeIndexer, cveIndexer),
		graphProvider: graphProvider,
	}
}
