package search

import (
	"context"

	"github.com/stackrox/rox/central/risk/datastore/internal/store"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/paginated"
)

var (
	defaultSortOption = &v1.QuerySortOption{
		Field:    search.RiskScore.String(),
		Reversed: true,
	}
)

// searcherImpl provides an intermediary implementation layer for RiskStorage.
type searcherImpl struct {
	storage  store.Store
	searcher search.Searcher
}

// SearchRawRisks retrieves Risks from the storage
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

// Format the search functionality for default sorting.
func formatSearcher(searcher search.Searcher) search.Searcher {
	defaultSortedSearcher := paginated.WithDefaultSortOption(searcher, defaultSortOption)
	return defaultSortedSearcher
}
