package search

import (
	"context"

	"github.com/stackrox/stackrox/central/risk/datastore/internal/index"
	"github.com/stackrox/stackrox/central/risk/datastore/internal/store"
	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/search"
)

// Searcher provides search functionality on existing risks
//go:generate mockgen-wrapper
type Searcher interface {
	Search(ctx context.Context, q *v1.Query) ([]search.Result, error)
	Count(ctx context.Context, q *v1.Query) (int, error)
	SearchRawRisks(ctx context.Context, q *v1.Query) ([]*storage.Risk, error)
}

// New returns a new instance of Searcher for the given storage and indexer.
func New(storage store.Store, indexer index.Indexer) Searcher {
	return &searcherImpl{
		storage:  storage,
		indexer:  indexer,
		searcher: formatSearcher(indexer),
	}
}
