package search

import (
	"context"

	podIndexer "github.com/stackrox/rox/central/pod/index"
	"github.com/stackrox/rox/central/pod/store"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
)

// Searcher provides search functionality on existing pods
//
//go:generate mockgen-wrapper
type Searcher interface {
	Search(ctx context.Context, q *v1.Query) ([]search.Result, error)
	Count(ctx context.Context, q *v1.Query) (int, error)
	SearchRawPods(ctx context.Context, q *v1.Query) ([]*storage.Pod, error)
}

// New returns a new instance of Searcher for the given storage and indexer.
func New(storage store.Store, podIndexer podIndexer.Indexer) Searcher {
	return &searcherImpl{
		storage:  storage,
		searcher: formatSearcher(podIndexer),
	}
}
