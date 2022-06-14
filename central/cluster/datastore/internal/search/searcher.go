package search

import (
	"context"

	"github.com/stackrox/stackrox/central/cluster/index"
	clusterStore "github.com/stackrox/stackrox/central/cluster/store/cluster"
	"github.com/stackrox/stackrox/central/ranking"
	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/dackbox/graph"
	"github.com/stackrox/stackrox/pkg/search"
)

// Searcher encapsulates cluster search functionality.
type Searcher interface {
	Search(ctx context.Context, q *v1.Query) ([]search.Result, error)
	Count(ctx context.Context, q *v1.Query) (int, error)
	SearchResults(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error)
	SearchClusters(ctx context.Context, q *v1.Query) ([]*storage.Cluster, error)
}

// New returns a new instance of Searcher for the given storage and indexer.
func New(storage clusterStore.Store, indexer index.Indexer, graphProvider graph.Provider, clusterRanker *ranking.Ranker) Searcher {
	return &searcherImpl{
		clusterStorage:    storage,
		indexer:           indexer,
		formattedSearcher: formatSearcher(indexer, graphProvider, clusterRanker),
	}
}
