package search

import (
	"context"

	"github.com/stackrox/rox/central/reportconfigurations/index"
	"github.com/stackrox/rox/central/reportconfigurations/store"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/blevesearch"
	"github.com/stackrox/rox/pkg/search/paginated"
)

var (
	defaultSortOption = &v1.QuerySortOption{
		Field: search.ReportName.String(),
	}
)

type searcherImpl struct {
	storage  store.Store
	searcher search.Searcher
	indexer  index.Indexer
}

func (s *searcherImpl) getSearchResults(ctx context.Context, q *v1.Query) ([]search.Result, error) {
	return s.searcher.Search(ctx, q)
}

// Search returns the raw search results from the query
func (s *searcherImpl) Search(ctx context.Context, q *v1.Query) ([]search.Result, error) {
	return s.getSearchResults(ctx, q)
}

func (s *searcherImpl) SearchReportConfigurations(ctx context.Context, query *v1.Query) ([]*storage.ReportConfiguration, error) {
	return s.searchReportConfigurations(ctx, query)
}

func (s *searcherImpl) Count(ctx context.Context, query *v1.Query) (int, error) {
	return s.searcher.Count(ctx, query)
}

// Format the search functionality of the indexer to be filtered (for sac) and paginated.
func formatSearcher(unsafeSearcher blevesearch.UnsafeSearcher) search.Searcher {
	safeSearcher := blevesearch.WrapUnsafeSearcherAsSearcher(unsafeSearcher) // Make the UnsafeSearcher safe.
	paginatedSearcher := paginated.Paginated(safeSearcher)
	defaultSortedSearcher := paginated.WithDefaultSortOption(paginatedSearcher, defaultSortOption)
	return defaultSortedSearcher
}

func (s *searcherImpl) searchReportConfigurations(ctx context.Context, q *v1.Query) ([]*storage.ReportConfiguration, error) {
	results, err := s.Search(ctx, q)
	if err != nil {
		return nil, err
	}

	ids := search.ResultsToIDs(results)
	reportConfigs, _, err := s.storage.GetMany(ctx, ids)
	if err != nil {
		return nil, err
	}
	return reportConfigs, nil
}
