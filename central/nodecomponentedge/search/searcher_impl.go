package search

import (
	"context"

	"github.com/stackrox/rox/central/nodecomponentedge/index"
	"github.com/stackrox/rox/central/nodecomponentedge/store"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
)

type searcherImpl struct {
	storage  store.Store
	indexer  index.Indexer
	searcher search.Searcher
}

// SearchEdges returns the search results from indexed edges for the query.
func (ds *searcherImpl) SearchEdges(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error) {
	results, err := ds.getSearchResults(ctx, q)
	if err != nil {
		return nil, err
	}
	return ds.resultsToSearchResults(ctx, results)
}

// Search returns the raw search results from the query
func (ds *searcherImpl) Search(ctx context.Context, q *v1.Query) ([]search.Result, error) {
	return ds.getSearchResults(ctx, q)
}

// Count returns the number of search results from the query
func (ds *searcherImpl) Count(ctx context.Context, q *v1.Query) (int, error) {
	return ds.searcher.Count(ctx, q)
}

// SearchRawEdges retrieves edges from the indexer and storage
func (ds *searcherImpl) SearchRawEdges(ctx context.Context, q *v1.Query) ([]*storage.NodeComponentEdge, error) {
	return ds.searchNodeComponentEdges(ctx, q)
}

func (ds *searcherImpl) getSearchResults(ctx context.Context, q *v1.Query) ([]search.Result, error) {
	return ds.searcher.Search(ctx, q)
}

// resultsToNodeComponentEdges returns the cves from the db for the given search results.
func (ds *searcherImpl) resultsToNodeComponentEdges(ctx context.Context, results []search.Result) ([]*storage.NodeComponentEdge, []int, error) {
	return ds.storage.GetMany(ctx, search.ResultsToIDs(results))
}

// resultsToSearchResults returns the searchResults from the db for the given search results.
func (ds *searcherImpl) resultsToSearchResults(ctx context.Context, results []search.Result) ([]*v1.SearchResult, error) {
	cves, missingIndices, err := ds.resultsToNodeComponentEdges(ctx, results)
	if err != nil {
		return nil, err
	}
	results = search.RemoveMissingResults(results, missingIndices)
	return convertMany(cves, results), nil
}

func convertMany(cves []*storage.NodeComponentEdge, results []search.Result) []*v1.SearchResult {
	outputResults := make([]*v1.SearchResult, len(cves))
	for idx, sar := range cves {
		outputResults[idx] = convertOne(sar, &results[idx])
	}
	return outputResults
}

func convertOne(cve *storage.NodeComponentEdge, result *search.Result) *v1.SearchResult {
	return &v1.SearchResult{
		Category:       v1.SearchCategory_NODE_COMPONENT_EDGE,
		Id:             cve.GetId(),
		Name:           cve.GetId(),
		FieldToMatches: search.GetProtoMatchesMap(result.Matches),
		Score:          result.Score,
	}
}

func (ds *searcherImpl) searchNodeComponentEdges(ctx context.Context, q *v1.Query) ([]*storage.NodeComponentEdge, error) {
	results, err := ds.Search(ctx, q)
	if err != nil {
		return nil, err
	}

	ids := search.ResultsToIDs(results)
	edges, _, err := ds.storage.GetMany(ctx, ids)
	if err != nil {
		return nil, err
	}
	return edges, nil
}
