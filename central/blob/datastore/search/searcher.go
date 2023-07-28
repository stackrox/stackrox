package search

import (
	"context"

	"github.com/stackrox/rox/central/blob/datastore/index"
	"github.com/stackrox/rox/central/blob/datastore/store"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	pkgSearch "github.com/stackrox/rox/pkg/search"
)

var (
	log = logging.LoggerForModule()
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
	ds := &searcherImpl{
		storage:           blobStore,
		formattedSearcher: formatSearcher(indexer),
	}
	return ds
}
