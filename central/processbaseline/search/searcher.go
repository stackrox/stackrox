package search

import (
	"context"

	"github.com/stackrox/rox/central/processbaseline/index"
	"github.com/stackrox/rox/central/processbaseline/store"
	"github.com/stackrox/rox/central/role/resources"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sac"
	pkgSearch "github.com/stackrox/rox/pkg/search"
)

var (
	log = logging.LoggerForModule()
)

// Searcher provides search functionality on existing alerts
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
		indexer:           indexer,
		formattedSearcher: formatSearcher(indexer),
	}

	ctx := sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.ProcessWhitelist)))
	if err := ds.buildIndex(ctx); err != nil {
		return nil, err
	}
	return ds, nil
}
