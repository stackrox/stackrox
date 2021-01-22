package search

import (
	"context"

	"github.com/stackrox/rox/central/nodecomponentedge/index"
	pkgNodeComponentEdgeSAC "github.com/stackrox/rox/central/nodecomponentedge/sac"
	"github.com/stackrox/rox/central/nodecomponentedge/store"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/blevesearch"
	"github.com/stackrox/rox/pkg/search/filtered"
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
	return ds.resultsToSearchResults(results)
}

// Search returns the raw search results from the query
func (ds *searcherImpl) Search(ctx context.Context, q *v1.Query) ([]search.Result, error) {
	return ds.getSearchResults(ctx, q)
}

// SearchRawEdges retrieves edges from the indexer and storage
func (ds *searcherImpl) SearchRawEdges(ctx context.Context, q *v1.Query) ([]*storage.NodeComponentEdge, error) {
	return ds.searchNodeComponentEdges(ctx, q)
}

func (ds *searcherImpl) getSearchResults(ctx context.Context, q *v1.Query) ([]search.Result, error) {
	return ds.searcher.Search(ctx, q)
}

// resultsToNodeComponentEdges returns the cves from the db for the given search results.
func (ds *searcherImpl) resultsToNodeComponentEdges(results []search.Result) ([]*storage.NodeComponentEdge, []int, error) {
	return ds.storage.GetBatch(search.ResultsToIDs(results))
}

// resultsToSearchResults returns the searchResults from the db for the given search results.
func (ds *searcherImpl) resultsToSearchResults(results []search.Result) ([]*v1.SearchResult, error) {
	cves, missingIndices, err := ds.resultsToNodeComponentEdges(results)
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

// Format the search functionality of the indexer to be filtered (for sac) and paginated.
func formatSearcher(unsafeSearcher blevesearch.UnsafeSearcher) search.Searcher {
	return filtered.UnsafeSearcher(unsafeSearcher, pkgNodeComponentEdgeSAC.GetSACFilter())
}

func (ds *searcherImpl) searchNodeComponentEdges(ctx context.Context, q *v1.Query) ([]*storage.NodeComponentEdge, error) {
	results, err := ds.Search(ctx, q)
	if err != nil {
		return nil, err
	}

	ids := search.ResultsToIDs(results)
	edges, _, err := ds.storage.GetBatch(ids)
	if err != nil {
		return nil, err
	}
	return edges, nil
}
