package search

import (
	"context"

	"github.com/stackrox/rox/central/imagecveedge/index"
	pkgImageCveEdgeSAC "github.com/stackrox/rox/central/imagecveedge/sac"
	"github.com/stackrox/rox/central/imagecveedge/store"
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

// SearchEdges returns the search results from indexed image-cve edges for the query.
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

// Count returns the number of search results from the query
func (ds *searcherImpl) Count(ctx context.Context, q *v1.Query) (int, error) {
	return ds.searcher.Count(ctx, q)
}

// SearchRawEdges retrieves image-cve edges from the indexer and storage
func (ds *searcherImpl) SearchRawEdges(ctx context.Context, q *v1.Query) ([]*storage.ImageCVEEdge, error) {
	return ds.searchImageCVEEdges(ctx, q)
}

func (ds *searcherImpl) getSearchResults(ctx context.Context, q *v1.Query) ([]search.Result, error) {
	return ds.searcher.Search(ctx, q)
}

// resultsToImageCVEEdges returns the ImageCVEEdges from the db for the given search results.
func (ds *searcherImpl) resultsToImageCVEEdges(results []search.Result) ([]*storage.ImageCVEEdge, []int, error) {
	return ds.storage.GetBatch(search.ResultsToIDs(results))
}

// ToSearchResults returns the searchResults from the db for the given search results.
func (ds *searcherImpl) resultsToSearchResults(results []search.Result) ([]*v1.SearchResult, error) {
	edges, missingIndices, err := ds.resultsToImageCVEEdges(results)
	if err != nil {
		return nil, err
	}
	results = search.RemoveMissingResults(results, missingIndices)
	return convertMany(edges, results), nil
}

func convertMany(edges []*storage.ImageCVEEdge, results []search.Result) []*v1.SearchResult {
	ret := make([]*v1.SearchResult, len(edges))
	for index, sar := range edges {
		ret[index] = convertOne(sar, &results[index])
	}
	return ret
}

func convertOne(cve *storage.ImageCVEEdge, result *search.Result) *v1.SearchResult {
	return &v1.SearchResult{
		Category:       v1.SearchCategory_IMAGE_VULN_EDGE,
		Id:             cve.GetId(),
		Name:           cve.GetId(),
		FieldToMatches: search.GetProtoMatchesMap(result.Matches),
		Score:          result.Score,
	}
}

// Format the search functionality of the indexer to be filtered (for sac) and paginated.
func formatSearcher(unsafeSearcher blevesearch.UnsafeSearcher) search.Searcher {
	return filtered.UnsafeSearcher(unsafeSearcher, pkgImageCveEdgeSAC.GetSACFilter())
}

func (ds *searcherImpl) searchImageCVEEdges(ctx context.Context, q *v1.Query) ([]*storage.ImageCVEEdge, error) {
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
