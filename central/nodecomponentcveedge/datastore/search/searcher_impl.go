package search

import (
	"context"

	"github.com/stackrox/rox/central/nodecomponentcveedge/datastore/index"
	pgStore "github.com/stackrox/rox/central/nodecomponentcveedge/datastore/store/postgres"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
)

type searcherImpl struct {
	storage  pgStore.Store
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
func (ds *searcherImpl) SearchRawEdges(ctx context.Context, q *v1.Query) ([]*storage.NodeComponentCVEEdge, error) {
	return ds.searchComponentCVEEdges(ctx, q)
}

func (ds *searcherImpl) resultsToEdges(ctx context.Context, results []search.Result) ([]*storage.NodeComponentCVEEdge, []int, error) {
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

func convertMany(cves []*storage.NodeComponentCVEEdge, results []search.Result) []*v1.SearchResult {
	outputResults := make([]*v1.SearchResult, len(cves))
	for index, sar := range cves {
		outputResults[index] = convertOne(sar, &results[index])
	}
	return outputResults
}

func convertOne(obj *storage.NodeComponentCVEEdge, result *search.Result) *v1.SearchResult {
	return &v1.SearchResult{
		Category:       v1.SearchCategory_NODE_COMPONENT_CVE_EDGE,
		Id:             obj.GetId(),
		Name:           obj.GetId(),
		FieldToMatches: search.GetProtoMatchesMap(result.Matches),
		Score:          result.Score,
	}
}

func (ds *searcherImpl) searchComponentCVEEdges(ctx context.Context, q *v1.Query) ([]*storage.NodeComponentCVEEdge, error) {
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
