package search

import (
	"context"

	"github.com/stackrox/stackrox/central/policy/index"
	"github.com/stackrox/stackrox/central/policy/store"
	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/logging"
	"github.com/stackrox/stackrox/pkg/search"
)

var (
	log = logging.LoggerForModule()
)

// Searcher provides search functionality on existing alerts
//go:generate mockgen-wrapper
type Searcher interface {
	Search(ctx context.Context, q *v1.Query) ([]search.Result, error)
	Count(ctx context.Context, q *v1.Query) (int, error)
	SearchPolicies(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error)
	SearchRawPolicies(ctx context.Context, q *v1.Query) ([]*storage.Policy, error)
}

// New returns a new instance of Searcher for the given storage and indexer.
func New(storage store.Store, indexer index.Indexer) Searcher {
	return &searcherImpl{
		storage:  storage,
		indexer:  indexer,
		searcher: formatSearcher(indexer),
	}
}
