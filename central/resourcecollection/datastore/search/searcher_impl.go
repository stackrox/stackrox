package search

import (
	"context"

	"github.com/stackrox/rox/central/resourcecollection/datastore/index"
	"github.com/stackrox/rox/central/resourcecollection/datastore/store/postgres"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
)

type searcherImpl struct {
	storage  postgres.Store
	indexer  index.Indexer
	searcher search.Searcher
}

func (s *searcherImpl) Search(ctx context.Context, q *v1.Query) ([]search.Result, error) {
	return s.getSearchResults(ctx, q)
}

// Count returns the number of search results from the query
func (s *searcherImpl) Count(ctx context.Context, q *v1.Query) (int, error) {
	return s.getCountResults(ctx, q)
}

func (s *searcherImpl) SearchCollections(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error) {
	results, err := s.getSearchResults(ctx, q)
	if err != nil {
		return nil, err
	}
	return s.resultsToSearchResults(ctx, results)
}

func (s *searcherImpl) SearchRawCollections(ctx context.Context, q *v1.Query) ([]*storage.ResourceCollection, error) {
	return s.SearchRawCollections(ctx, q)
}

func (s *searcherImpl) searchCollections(ctx context.Context, q *v1.Query) ([]*storage.ResourceCollection, error) {
	results, err := s.getSearchResults(ctx, q)
	if err != nil {
		return nil, err
	}

	ids := search.ResultsToIDs(results)
	collections, _, err := s.storage.GetMany(ctx, ids)
	if err != nil {
		return nil, err
	}
	return collections, nil
}

func (s *searcherImpl) getSearchResults(ctx context.Context, q *v1.Query) (res []search.Result, err error) {
	return s.searcher.Search(ctx, q)
}

func (s *searcherImpl) getCountResults(ctx context.Context, q *v1.Query) (count int, err error) {
	return s.searcher.Count(ctx, q)
}

func (s *searcherImpl) resultsToCollections(ctx context.Context, results []search.Result) ([]*storage.ResourceCollection, []int, error) {
	return s.storage.GetMany(ctx, search.ResultsToIDs(results))
}

func (s *searcherImpl) resultsToSearchResults(ctx context.Context, results []search.Result) ([]*v1.SearchResult, error) {
	collections, missingIndices, err := s.resultsToCollections(ctx, results)
	if err != nil {
		return nil, err
	}
	results = search.RemoveMissingResults(results, missingIndices)
	return convertMany(collections, results), nil
}

func convertMany(collections []*storage.ResourceCollection, results []search.Result) []*v1.SearchResult {
	outputResults := make([]*v1.SearchResult, len(collections))
	for idx, sar := range collections {
		outputResults[idx] = convertOne(sar, &results[idx])
	}
	return outputResults
}

func convertOne(collection *storage.ResourceCollection, result *search.Result) *v1.SearchResult {
	return &v1.SearchResult{
		Category:       v1.SearchCategory_COLLECTIONS,
		Id:             collection.GetId(),
		Name:           collection.GetName(),
		FieldToMatches: search.GetProtoMatchesMap(result.Matches),
		Score:          result.Score,
	}
}
