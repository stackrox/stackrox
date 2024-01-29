package search

import (
	"context"

	"github.com/stackrox/rox/central/blob/datastore/index"
	"github.com/stackrox/rox/central/blob/datastore/store"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	pkgSearch "github.com/stackrox/rox/pkg/search"
)

// Searcher provides search functionality on existing blobs
//
//go:generate mockgen-wrapper
type Searcher interface {
	Search(ctx context.Context, q *v1.Query) ([]pkgSearch.Result, error)
	Count(ctx context.Context, q *v1.Query) (int, error)
	SearchIDs(ctx context.Context, q *v1.Query) ([]string, error)
	SearchMetadata(ctx context.Context, q *v1.Query) ([]*storage.Blob, error)
}

// New returns a new instance of Searcher for the given storage and indexer.
func New(blobStore store.Store, indexer index.Indexer) Searcher {
	return &searcherImpl{
		storage:           blobStore,
		formattedSearcher: formatSearcher(indexer),
	}
}
