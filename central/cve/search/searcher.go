package search

import (
	"context"

	"github.com/stackrox/rox/central/cve/index"
	"github.com/stackrox/rox/central/cve/store"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
)

// Searcher provides search functionality on existing cves.
//go:generate mockgen-wrapper
type Searcher interface {
	Search(ctx context.Context, query *v1.Query) ([]search.Result, error)
	SearchCVEs(context.Context, *v1.Query) ([]*v1.SearchResult, error)
	SearchRawCVEs(ctx context.Context, query *v1.Query) ([]*storage.CVE, error)
}

// New returns a new instance of Searcher for the given storage and index.
func New(storage store.Store, indexer index.Indexer) Searcher {
	return &searcherImpl{
		storage:  storage,
		indexer:  indexer,
		searcher: formatSearcher(indexer),
	}
}
