package search

import (
	"context"

	"github.com/stackrox/rox/central/policycategoryedge/index"
	"github.com/stackrox/rox/central/policycategoryedge/store"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/search"
	pkgPostgres "github.com/stackrox/rox/pkg/search/scoped/postgres"
	"github.com/stackrox/rox/pkg/search/sortfields"
)

// New returns a new instance of Searcher for the given storage and indexer.
func New(storage store.Store, indexer index.Indexer) Searcher {
	return &searcherImpl{
		storage:  storage,
		indexer:  indexer,
		searcher: formatSearcherV2(indexer),
	}
}

func formatSearcherV2(searcher search.Searcher) search.Searcher {
	scopedSafeSearcher := pkgPostgres.WithScoping(searcher)
	return sortfields.TransformSortFields(scopedSafeSearcher, schema.CollectionsSchema.OptionsMap)
}

type searcherImpl struct {
	storage  store.Store
	indexer  index.Indexer
	searcher search.Searcher
}

// SearchEdges returns the search results from indexed edges for the query.
func (ds *searcherImpl) SearchEdges(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error) {
	results, err := ds.Search(ctx, q)
	if err != nil {
		return nil, err
	}
	return ds.resultsToSearchResults(ctx, results)
}

// Search returns the raw search results from the query
func (ds *searcherImpl) Search(ctx context.Context, q *v1.Query) (res []search.Result, err error) {
	return ds.searcher.Search(ctx, q)
}

// Count returns the number of search results from the query
func (ds *searcherImpl) Count(ctx context.Context, q *v1.Query) (count int, err error) {
	return ds.searcher.Count(ctx, q)
}

// SearchRawEdges retrieves edges from the indexer and storage
func (ds *searcherImpl) SearchRawEdges(ctx context.Context, q *v1.Query) ([]*storage.PolicyCategoryEdge, error) {
	return ds.searchPolicyCategoryEdges(ctx, q)
}

func (ds *searcherImpl) resultsToEdges(ctx context.Context, results []search.Result) ([]*storage.PolicyCategoryEdge, []int, error) {
	return ds.storage.GetMany(ctx, search.ResultsToIDs(results))
}

func (ds *searcherImpl) resultsToSearchResults(ctx context.Context, results []search.Result) ([]*v1.SearchResult, error) {
	cves, missingIndices, err := ds.resultsToEdges(ctx, results)
	if err != nil {
		return nil, err
	}
	results = search.RemoveMissingResults(results, missingIndices)
	return convertMany(cves, results), nil
}

func convertMany(cves []*storage.PolicyCategoryEdge, results []search.Result) []*v1.SearchResult {
	outputResults := make([]*v1.SearchResult, len(cves))
	for index, sar := range cves {
		outputResults[index] = convertOne(sar, &results[index])
	}
	return outputResults
}

func convertOne(obj *storage.PolicyCategoryEdge, result *search.Result) *v1.SearchResult {
	return &v1.SearchResult{
		Category:       v1.SearchCategory_POLICY_CATEGORY_EDGE,
		Id:             obj.GetId(),
		Name:           obj.GetId(),
		FieldToMatches: search.GetProtoMatchesMap(result.Matches),
		Score:          result.Score,
	}
}

func (ds *searcherImpl) searchPolicyCategoryEdges(ctx context.Context, q *v1.Query) ([]*storage.PolicyCategoryEdge, error) {
	results, err := ds.Search(ctx, q)
	if err != nil {
		return nil, err
	}

	ids := search.ResultsToIDs(results)
	cves, _, err := ds.storage.GetMany(ctx, ids)
	if err != nil {
		return nil, err
	}
	return cves, nil
}
