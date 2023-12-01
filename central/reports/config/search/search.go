package search

import (
	"context"

	"github.com/stackrox/rox/central/reports/config/index"
	"github.com/stackrox/rox/central/reports/config/store"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
)

// Searcher provides search functionality on existing report configurations.
//
//go:generate mockgen-wrapper
type Searcher interface {
	Search(ctx context.Context, query *v1.Query) ([]search.Result, error)
	SearchReportConfigurations(ctx context.Context, query *v1.Query) ([]*storage.ReportConfiguration, error)
	Count(ctx context.Context, query *v1.Query) (int, error)
}

// New returns a new instance of Searcher for the given storage and index.
func New(storage store.Store, indexer index.Indexer) *searcherImpl {
	return &searcherImpl{
		storage:  storage,
		indexer:  indexer,
		searcher: formatSearcher(indexer),
	}
}
