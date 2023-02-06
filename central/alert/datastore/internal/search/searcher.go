package search

import (
	"context"

	"github.com/stackrox/rox/central/alert/datastore/internal/index"
	"github.com/stackrox/rox/central/alert/datastore/internal/store"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
)

// Searcher provides search functionality on existing alerts
//
//go:generate mockgen-wrapper
type Searcher interface {
	SearchAlerts(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error)
	SearchRawAlerts(ctx context.Context, q *v1.Query) ([]*storage.Alert, error)
	SearchListAlerts(ctx context.Context, q *v1.Query) ([]*storage.ListAlert, error)
	Search(ctx context.Context, q *v1.Query) ([]search.Result, error)
	Count(ctx context.Context, q *v1.Query) (int, error)
}

// New returns a new instance of Searcher for the given storage and indexer.
func New(storage store.Store, indexer index.Indexer) Searcher {
	return &searcherImpl{
		storage:           storage,
		indexer:           indexer,
		formattedSearcher: formatSearcher(indexer),
	}
}
