package search

import (
	"context"

	pgStore "github.com/stackrox/rox/central/imagecomponent/v2/datastore/store/postgres"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/paginated"
	"github.com/stackrox/rox/pkg/search/scoped/postgres"
	"github.com/stackrox/rox/pkg/search/sortfields"
)

var (
	defaultSortOption = &v1.QuerySortOption{
		Field: search.Component.String(),
	}
)

// NewV2 returns a new instance of Searcher for the given store.
func NewV2(storage pgStore.Store) Searcher {
	return &searcherImplV2{
		storage:  storage,
		searcher: formatSearcherV2(storage),
	}
}

func formatSearcherV2(searcher search.Searcher) search.Searcher {
	scopedSearcher := postgres.WithScoping(searcher)
	transformedSortFieldSearcher := sortfields.TransformSortFields(scopedSearcher, schema.ImagesSchema.OptionsMap)
	return paginated.WithDefaultSortOption(transformedSortFieldSearcher, defaultSortOption)
}

type searcherImplV2 struct {
	storage  pgStore.Store
	searcher search.Searcher
}

func (s *searcherImplV2) Search(ctx context.Context, q *v1.Query) ([]search.Result, error) {
	return s.getSearchResults(ctx, q)
}

// Count returns the number of search results from the query
func (s *searcherImplV2) Count(ctx context.Context, q *v1.Query) (int, error) {
	return s.getCountResults(ctx, q)
}

func (s *searcherImplV2) SearchImageComponents(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error) {
	results, err := s.getSearchResults(ctx, q)
	if err != nil {
		return nil, err
	}
	return s.resultsToSearchResults(ctx, results)
}

func (s *searcherImplV2) SearchRawImageComponents(ctx context.Context, q *v1.Query) ([]*storage.ImageComponentV2, error) {
	return s.searchImageComponents(ctx, q)
}

func (s *searcherImplV2) searchImageComponents(ctx context.Context, q *v1.Query) ([]*storage.ImageComponentV2, error) {
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

func (s *searcherImplV2) getSearchResults(ctx context.Context, q *v1.Query) (res []search.Result, err error) {
	return s.searcher.Search(ctx, q)
}

func (s *searcherImplV2) getCountResults(ctx context.Context, q *v1.Query) (count int, err error) {
	return s.searcher.Count(ctx, q)
}

func (s *searcherImplV2) resultsToImageComponents(ctx context.Context, results []search.Result) ([]*storage.ImageComponentV2, []int, error) {
	return s.storage.GetMany(ctx, search.ResultsToIDs(results))
}

func (s *searcherImplV2) resultsToSearchResults(ctx context.Context, results []search.Result) ([]*v1.SearchResult, error) {
	components, missingIndices, err := s.resultsToImageComponents(ctx, results)
	if err != nil {
		return nil, err
	}
	results = search.RemoveMissingResults(results, missingIndices)
	return convertMany(components, results), nil
}

func convertMany(components []*storage.ImageComponentV2, results []search.Result) []*v1.SearchResult {
	outputResults := make([]*v1.SearchResult, len(components))
	for index, sar := range components {
		outputResults[index] = convertOne(sar, &results[index])
	}
	return outputResults
}

func convertOne(component *storage.ImageComponentV2, result *search.Result) *v1.SearchResult {
	return &v1.SearchResult{
		Category:       v1.SearchCategory_IMAGE_COMPONENTS_V2,
		Id:             component.GetId(),
		Name:           component.GetName(),
		FieldToMatches: search.GetProtoMatchesMap(result.Matches),
		Score:          result.Score,
	}
}
