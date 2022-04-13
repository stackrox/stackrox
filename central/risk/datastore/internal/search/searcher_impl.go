package search

import (
	"context"

	"github.com/stackrox/stackrox/central/risk/datastore/internal/index"
	"github.com/stackrox/stackrox/central/risk/datastore/internal/store"
	"github.com/stackrox/stackrox/central/risk/mappings"
	"github.com/stackrox/stackrox/central/role/resources"
	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/sac"
	"github.com/stackrox/stackrox/pkg/search"
	"github.com/stackrox/stackrox/pkg/search/blevesearch"
	"github.com/stackrox/stackrox/pkg/search/paginated"
)

var (
	defaultSortOption = &v1.QuerySortOption{
		Field:    search.RiskScore.String(),
		Reversed: true,
	}

	riskSACSearchHelper = sac.ForResource(resources.Risk).MustCreateSearchHelper(mappings.OptionsMap)
)

// searcherImpl provides an intermediary implementation layer for RiskStorage.
type searcherImpl struct {
	storage  store.Store
	indexer  index.Indexer
	searcher search.Searcher
}

// SearchRawRisks retrieves Risks from the indexer and storage
func (s *searcherImpl) SearchRawRisks(ctx context.Context, q *v1.Query) ([]*storage.Risk, error) {
	results, err := s.Search(ctx, q)
	if err != nil {
		return nil, err
	}

	ids := search.ResultsToIDs(results)
	risks, _, err := s.storage.GetMany(ctx, ids)
	if err != nil {
		return nil, err
	}
	return risks, nil
}

func (s *searcherImpl) Search(ctx context.Context, q *v1.Query) ([]search.Result, error) {
	return s.searcher.Search(ctx, q)
}

// Count returns the number of search results from the query
func (s *searcherImpl) Count(ctx context.Context, q *v1.Query) (int, error) {
	return s.searcher.Count(ctx, q)
}

// Format the search functionality of the indexer to be filtered (for sac) and paginated.
func formatSearcher(unsafeSearcher blevesearch.UnsafeSearcher) search.Searcher {
	filteredSearcher := riskSACSearchHelper.FilteredSearcher(unsafeSearcher) // Make the UnsafeSearcher safe.
	paginatedSearcher := paginated.Paginated(filteredSearcher)
	defaultSortedSearcher := paginated.WithDefaultSortOption(paginatedSearcher, defaultSortOption)
	return defaultSortedSearcher
}
