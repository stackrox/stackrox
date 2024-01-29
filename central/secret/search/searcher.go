package search

import (
	"context"

	"github.com/stackrox/rox/central/secret/internal/index"
	"github.com/stackrox/rox/central/secret/internal/store"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
)

// Searcher provides search functionality on existing secrets.
//
//go:generate mockgen-wrapper
type Searcher interface {
	Search(ctx context.Context, query *v1.Query) ([]search.Result, error)
	Count(ctx context.Context, query *v1.Query) (int, error)
	SearchSecrets(context.Context, *v1.Query) ([]*v1.SearchResult, error)
	SearchListSecrets(ctx context.Context, query *v1.Query) ([]*storage.ListSecret, error)
	SearchRawSecrets(ctx context.Context, query *v1.Query) ([]*storage.Secret, error)
}

// New returns a new instance of Searcher for the given storage and index.
func New(storage store.Store, indexer index.Indexer) Searcher {
	return &searcherImpl{
		storage:  storage,
		indexer:  indexer,
		searcher: formatSearcher(indexer),
	}
}
