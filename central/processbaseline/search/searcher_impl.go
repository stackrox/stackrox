package search

import (
	"context"

	"github.com/stackrox/rox/central/processbaseline/index"
	"github.com/stackrox/rox/central/processbaseline/store"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/paginated"
)

var (
	deploymentExtensionPostgresSACSearchHelper = sac.ForResource(resources.DeploymentExtension).MustCreatePgSearchHelper()
)

type searcherImpl struct {
	storage           store.Store
	indexer           index.Indexer
	formattedSearcher search.Searcher
}

func (s *searcherImpl) SearchRawProcessBaselines(ctx context.Context, q *v1.Query) ([]*storage.ProcessBaseline, error) {
	var (
		results []search.Result
		err     error
	)
	results, err = deploymentExtensionPostgresSACSearchHelper.FilteredSearcher(s.indexer).Search(ctx, q)

	if err != nil || len(results) == 0 {
		return nil, err
	}
	ids := search.ResultsToIDs(results)
	baselines, _, err := s.storage.GetMany(ctx, ids)
	if err != nil {
		return nil, err
	}
	return baselines, nil
}

func (s *searcherImpl) Search(ctx context.Context, q *v1.Query) ([]search.Result, error) {
	return s.formattedSearcher.Search(ctx, q)
}

// Count returns the number of search results from the query
func (s *searcherImpl) Count(ctx context.Context, q *v1.Query) (int, error) {
	return s.formattedSearcher.Count(ctx, q)
}

// Helper functions which format our searching.
///////////////////////////////////////////////

func formatSearcher(searcher search.Searcher) search.Searcher {
	filteredSearcher := deploymentExtensionPostgresSACSearchHelper.FilteredSearcher(searcher) // Make the
	// UnsafeSearcher safe.

	paginatedSearcher := paginated.Paginated(filteredSearcher)
	return paginatedSearcher
}
