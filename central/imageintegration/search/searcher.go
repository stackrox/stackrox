package search

import (
	"context"

	"github.com/stackrox/rox/central/imageintegration/index"
	"github.com/stackrox/rox/central/imageintegration/store"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/search"
)

// Searcher provides search functionality on existing image integrations
//
//go:generate mockgen-wrapper
type Searcher interface {
	Search(ctx context.Context, q *v1.Query) ([]search.Result, error)
	Count(ctx context.Context, q *v1.Query) (int, error)
	SearchImageIntegrations(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error)
}

// New returns a new instance of Searcher for the given storage and indexer.
func New(storage store.Store, indexer index.Indexer) Searcher {
	return &searcherImpl{
		storage:  storage,
		searcher: indexer,
	}
}
