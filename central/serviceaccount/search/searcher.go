package search

import (
	"context"

	"github.com/stackrox/rox/central/serviceaccount/internal/index"
	"github.com/stackrox/rox/central/serviceaccount/internal/store"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
)

// Searcher provides search functionality on existing service accounts.
//
//go:generate mockgen-wrapper
type Searcher interface {
	Search(ctx context.Context, query *v1.Query) ([]search.Result, error)
	Count(ctx context.Context, query *v1.Query) (int, error)
	SearchServiceAccounts(context.Context, *v1.Query) ([]*v1.SearchResult, error)
	SearchRawServiceAccounts(context.Context, *v1.Query) ([]*storage.ServiceAccount, error)
}

// New returns a new instance of Searcher for the given storage and index.
func New(storage store.Store, indexer index.Indexer) Searcher {
	return &searcherImpl{
		storage: storage,
		indexer: indexer,
	}
}
