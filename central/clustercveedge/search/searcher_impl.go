package search

import (
	"context"

	"github.com/stackrox/rox/central/clustercveedge/index"
	pkgClusterCVEEdgeSAC "github.com/stackrox/rox/central/clustercveedge/sac"
	"github.com/stackrox/rox/central/clustercveedge/store"
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

// SearchClusterCVEEdges returns the search results from indexed cves for the query.
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

// SearchRawClusterCVEEdges retrieves cves from the indexer and storage
func (ds *searcherImpl) SearchRawEdges(ctx context.Context, q *v1.Query) ([]*storage.ClusterCVEEdge, error) {
	return ds.searchClusterCVEEdges(ctx, q)
}

func (ds *searcherImpl) getSearchResults(ctx context.Context, q *v1.Query) ([]search.Result, error) {
	return ds.searcher.Search(ctx, q)
}

// ToClusterCVEEdges returns the cves from the db for the given search results.
func (ds *searcherImpl) resultsToListClusterCVEEdges(results []search.Result) ([]*storage.ClusterCVEEdge, []int, error) {
	return ds.storage.GetBatch(search.ResultsToIDs(results))
}

// ToSearchResults returns the searchResults from the db for the given search results.
func (ds *searcherImpl) resultsToSearchResults(results []search.Result) ([]*v1.SearchResult, error) {
	cves, missingIndices, err := ds.resultsToListClusterCVEEdges(results)
	if err != nil {
		return nil, err
	}
	results = search.RemoveMissingResults(results, missingIndices)
	return convertMany(cves, results), nil
}

func convertMany(cves []*storage.ClusterCVEEdge, results []search.Result) []*v1.SearchResult {
	outputResults := make([]*v1.SearchResult, len(cves))
	for index, sar := range cves {
		outputResults[index] = convertOne(sar, &results[index])
	}
	return outputResults
}

func convertOne(cve *storage.ClusterCVEEdge, result *search.Result) *v1.SearchResult {
	return &v1.SearchResult{
		Category:       v1.SearchCategory_CLUSTER_VULN_EDGE,
		Id:             cve.GetId(),
		Name:           cve.GetId(),
		FieldToMatches: search.GetProtoMatchesMap(result.Matches),
		Score:          result.Score,
	}
}

// Format the search functionality of the indexer to be filtered (for sac) and paginated.
func formatSearcher(unsafeSearcher blevesearch.UnsafeSearcher) search.Searcher {
	return filtered.UnsafeSearcher(unsafeSearcher, pkgClusterCVEEdgeSAC.GetSACFilter())
}

func (ds *searcherImpl) searchClusterCVEEdges(ctx context.Context, q *v1.Query) ([]*storage.ClusterCVEEdge, error) {
	results, err := ds.Search(ctx, q)
	if err != nil {
		return nil, err
	}

	ids := search.ResultsToIDs(results)
	cves, _, err := ds.storage.GetBatch(ids)
	if err != nil {
		return nil, err
	}
	return cves, nil
}
