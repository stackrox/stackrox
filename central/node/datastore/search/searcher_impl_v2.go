package search

import (
	"context"

	"github.com/stackrox/rox/central/node/datastore/store"
	"github.com/stackrox/rox/central/node/index"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/paginated"
	"github.com/stackrox/rox/pkg/search/scoped/postgres"
	"github.com/stackrox/rox/pkg/search/sortfields"
)

var (
	sacHelper         = sac.ForResource(resources.Node).MustCreatePgSearchHelper()
	defaultSortOption = &v1.QuerySortOption{
		Field: search.LastUpdatedTime.String(),
	}
)

// NewV2 returns a new instance of Searcher for the given storage and indexer.
func NewV2(storage store.Store, indexer index.Indexer) Searcher {
	return &searcherImplV2{
		storage:  storage,
		indexer:  indexer,
		searcher: formatSearcherV2(indexer),
	}
}

func formatSearcherV2(searcher search.Searcher) search.Searcher {
	safeSearcher := sacHelper.FilteredSearcher(searcher)
	scopedSafeSearcher := postgres.WithScoping(safeSearcher)
	transformedSortFieldSearcher := sortfields.TransformSortFields(scopedSafeSearcher, schema.NodesSchema.OptionsMap)
	return paginated.WithDefaultSortOption(transformedSortFieldSearcher, defaultSortOption)
}

type searcherImplV2 struct {
	storage  store.Store
	indexer  index.Indexer
	searcher search.Searcher
}

func (s *searcherImplV2) Search(ctx context.Context, q *v1.Query) ([]search.Result, error) {
	return s.getSearchResults(ctx, q)
}

// Count returns the number of search results from the query
func (s *searcherImplV2) Count(ctx context.Context, q *v1.Query) (int, error) {
	return s.getCountResults(ctx, q)
}

func (s *searcherImplV2) SearchNodes(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error) {
	results, err := s.getSearchResults(ctx, q)
	if err != nil {
		return nil, err
	}
	return s.resultsToSearchResults(ctx, results)
}

func (s *searcherImplV2) SearchRawNodes(ctx context.Context, q *v1.Query) ([]*storage.Node, error) {
	return s.searchNodes(ctx, q)
}

func (s *searcherImplV2) searchNodes(ctx context.Context, q *v1.Query) ([]*storage.Node, error) {
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

func (s *searcherImplV2) resultsToNodes(ctx context.Context, results []search.Result) ([]*storage.Node, []int, error) {
	return s.storage.GetMany(ctx, search.ResultsToIDs(results))
}

func (s *searcherImplV2) resultsToSearchResults(ctx context.Context, results []search.Result) ([]*v1.SearchResult, error) {
	components, missingIndices, err := s.resultsToNodes(ctx, results)
	if err != nil {
		return nil, err
	}
	results = search.RemoveMissingResults(results, missingIndices)
	return convertMany(components, results), nil
}

func convertMany(components []*storage.Node, results []search.Result) []*v1.SearchResult {
	outputResults := make([]*v1.SearchResult, len(components))
	for i, sar := range components {
		outputResults[i] = convertOne(sar, &results[i])
	}
	return outputResults
}

func convertOne(node *storage.Node, result *search.Result) *v1.SearchResult {
	return &v1.SearchResult{
		Category:       v1.SearchCategory_NODES,
		Id:             node.GetId(),
		Name:           node.GetName(),
		FieldToMatches: search.GetProtoMatchesMap(result.Matches),
		Score:          result.Score,
	}
}
