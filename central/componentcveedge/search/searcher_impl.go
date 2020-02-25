package search

import (
	"context"

	"github.com/stackrox/rox/central/componentcveedge/index"
	pkgComponentCVEEdgeSAC "github.com/stackrox/rox/central/componentcveedge/sac"
	"github.com/stackrox/rox/central/componentcveedge/store"
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

// SearchComponentCVEEdges returns the search results from indexed cves for the query.
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

// SearchRawComponentCVEEdges retrieves cves from the indexer and storage
func (ds *searcherImpl) SearchRawEdges(ctx context.Context, q *v1.Query) ([]*storage.ComponentCVEEdge, error) {
	return ds.searchComponentCVEEdges(ctx, q)
}

func (ds *searcherImpl) getSearchResults(ctx context.Context, q *v1.Query) ([]search.Result, error) {
	return ds.searcher.Search(ctx, q)
}

// ToComponentCVEEdges returns the cves from the db for the given search results.
func (ds *searcherImpl) resultsToListComponentCVEEdges(results []search.Result) ([]*storage.ComponentCVEEdge, []int, error) {
	return ds.storage.GetBatch(search.ResultsToIDs(results))
}

// ToSearchResults returns the searchResults from the db for the given search results.
func (ds *searcherImpl) resultsToSearchResults(results []search.Result) ([]*v1.SearchResult, error) {
	cves, missingIndices, err := ds.resultsToListComponentCVEEdges(results)
	if err != nil {
		return nil, err
	}
	results = search.RemoveMissingResults(results, missingIndices)
	return convertMany(cves, results), nil
}

func convertMany(cves []*storage.ComponentCVEEdge, results []search.Result) []*v1.SearchResult {
	outputResults := make([]*v1.SearchResult, len(cves))
	for index, sar := range cves {
		outputResults[index] = convertOne(sar, &results[index])
	}
	return outputResults
}

func convertOne(cve *storage.ComponentCVEEdge, result *search.Result) *v1.SearchResult {
	return &v1.SearchResult{
		Category:       v1.SearchCategory_COMPONENT_VULN_EDGE,
		Id:             cve.GetId(),
		Name:           cve.GetId(),
		FieldToMatches: search.GetProtoMatchesMap(result.Matches),
		Score:          result.Score,
	}
}

// Format the search functionality of the indexer to be filtered (for sac) and paginated.
func formatSearcher(unsafeSearcher blevesearch.UnsafeSearcher) search.Searcher {
	return filtered.UnsafeSearcher(unsafeSearcher, pkgComponentCVEEdgeSAC.GetSACFilter())
}

func (ds *searcherImpl) searchComponentCVEEdges(ctx context.Context, q *v1.Query) ([]*storage.ComponentCVEEdge, error) {
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
