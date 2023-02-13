package search

import (
	"context"

	"github.com/stackrox/rox/central/nodecomponent/datastore/index"
	pgStore "github.com/stackrox/rox/central/nodecomponent/datastore/store/postgres"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
)

type searcherImpl struct {
	storage  pgStore.Store
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

func (s *searcherImpl) SearchNodeComponents(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error) {
	results, err := s.getSearchResults(ctx, q)
	if err != nil {
		return nil, err
	}
	return s.resultsToSearchResults(ctx, results)
}

func (s *searcherImpl) SearchRawNodeComponents(ctx context.Context, q *v1.Query) ([]*storage.NodeComponent, error) {
	return s.searchNodeComponents(ctx, q)
}

func (s *searcherImpl) searchNodeComponents(ctx context.Context, q *v1.Query) ([]*storage.NodeComponent, error) {
	results, err := s.getSearchResults(ctx, q)
	if err != nil {
		return nil, err
	}

	ids := search.ResultsToIDs(results)
	components, _, err := s.storage.GetMany(ctx, ids)
	if err != nil {
		return nil, err
	}
	return components, nil
}

func (s *searcherImpl) getSearchResults(ctx context.Context, q *v1.Query) (res []search.Result, err error) {
	return s.searcher.Search(ctx, q)
}

func (s *searcherImpl) getCountResults(ctx context.Context, q *v1.Query) (count int, err error) {
	return s.searcher.Count(ctx, q)
}

func (s *searcherImpl) resultsToImageComponents(ctx context.Context, results []search.Result) ([]*storage.NodeComponent, []int, error) {
	return s.storage.GetMany(ctx, search.ResultsToIDs(results))
}

func (s *searcherImpl) resultsToSearchResults(ctx context.Context, results []search.Result) ([]*v1.SearchResult, error) {
	components, missingIndices, err := s.resultsToImageComponents(ctx, results)
	if err != nil {
		return nil, err
	}
	results = search.RemoveMissingResults(results, missingIndices)
	return convertMany(components, results), nil
}

func convertMany(components []*storage.NodeComponent, results []search.Result) []*v1.SearchResult {
	outputResults := make([]*v1.SearchResult, len(components))
	for index, sar := range components {
		outputResults[index] = convertOne(sar, &results[index])
	}
	return outputResults
}

func convertOne(component *storage.NodeComponent, result *search.Result) *v1.SearchResult {
	return &v1.SearchResult{
		Category:       v1.SearchCategory_NODE_COMPONENTS,
		Id:             component.GetId(),
		Name:           component.GetName(),
		FieldToMatches: search.GetProtoMatchesMap(result.Matches),
		Score:          result.Score,
	}
}
