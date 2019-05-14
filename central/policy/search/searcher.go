package search

import (
	"context"

	"github.com/stackrox/rox/central/policy/index"
	"github.com/stackrox/rox/central/policy/store"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/search"
)

var (
	log = logging.LoggerForModule()
)

// Searcher provides search functionality on existing alerts
type Searcher interface {
	Search(ctx context.Context, q *v1.Query) ([]search.Result, error)
	SearchPolicies(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error)
	SearchRawPolicies(ctx context.Context, q *v1.Query) ([]*storage.Policy, error)
}

// New returns a new instance of Searcher for the given storage and indexer.
func New(storage store.Store, indexer index.Indexer) (Searcher, error) {
	ds := &searcherImpl{
		storage: storage,
		indexer: indexer,
	}
	if err := ds.buildIndex(); err != nil {
		return nil, err
	}
	return ds, nil
}
