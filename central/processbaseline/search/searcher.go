package search

import (
	"context"

	"github.com/stackrox/rox/central/processbaseline/index"
	"github.com/stackrox/rox/central/processbaseline/store"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	pkgSearch "github.com/stackrox/rox/pkg/search"
)

// Searcher provides search functionality on existing alerts
//
//go:generate mockgen-wrapper
type Searcher interface {
	SearchRawProcessBaselines(ctx context.Context, q *v1.Query) ([]*storage.ProcessBaseline, error)
	Search(ctx context.Context, q *v1.Query) ([]pkgSearch.Result, error)
	Count(ctx context.Context, q *v1.Query) (int, error)
}

// New returns a new instance of Searcher for the given storage and indexer.
func New(processBaselineStore store.Store, indexer index.Indexer) (Searcher, error) {
	ds := &searcherImpl{
		storage:           processBaselineStore,
		formattedSearcher: formatSearcher(indexer),
	}

	return ds, nil
}
