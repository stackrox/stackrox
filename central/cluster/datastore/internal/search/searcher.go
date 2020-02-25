package search

import (
	"context"

	"github.com/stackrox/rox/central/cluster/index"
	"github.com/stackrox/rox/central/cluster/store"
	"github.com/stackrox/rox/central/ranking"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/dackbox/graph"
	"github.com/stackrox/rox/pkg/search"
)

// Searcher encapsulates cluster search functionality.
type Searcher interface {
	Search(ctx context.Context, q *v1.Query) ([]search.Result, error)
	SearchResults(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error)
	SearchClusters(ctx context.Context, q *v1.Query) ([]*storage.Cluster, error)
}

// New returns a new instance of Searcher for the given storage and indexer.
func New(storage store.Store, indexer index.Indexer, graphProvider graph.Provider, clusterRanker *ranking.Ranker) Searcher {
	return &searcherImpl{
		storage:           storage,
		indexer:           indexer,
		formattedSearcher: formatSearcher(indexer, graphProvider, clusterRanker),
	}
}
