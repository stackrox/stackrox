package search

import (
	"context"

	"github.com/stackrox/stackrox/central/reportconfigurations/index"
	"github.com/stackrox/stackrox/central/reportconfigurations/store"
	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/logging"
	"github.com/stackrox/stackrox/pkg/search"
)

var (
	log = logging.LoggerForModule()
)

// Searcher provides search functionality on existing report configurations.
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
