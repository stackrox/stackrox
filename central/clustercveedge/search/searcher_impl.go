package search

import (
	"context"

	"github.com/stackrox/stackrox/central/clustercveedge/index"
	clusterCVEEdgeMappings "github.com/stackrox/stackrox/central/clustercveedge/mappings"
	clusterCVEEdgeSAC "github.com/stackrox/stackrox/central/clustercveedge/sac"
	"github.com/stackrox/stackrox/central/clustercveedge/store"
	cveMappings "github.com/stackrox/stackrox/central/cve/mappings"
	"github.com/stackrox/stackrox/central/dackbox"
	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/dackbox/graph"
	"github.com/stackrox/stackrox/pkg/search"
	"github.com/stackrox/stackrox/pkg/search/blevesearch"
	"github.com/stackrox/stackrox/pkg/search/compound"
	"github.com/stackrox/stackrox/pkg/search/filtered"
	"github.com/stackrox/stackrox/pkg/search/scoped"
)

type searcherImpl struct {
	storage       store.Store
	indexer       index.Indexer
	searcher      search.Searcher
	graphProvider graph.Provider
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
func (ds *searcherImpl) Search(ctx context.Context, q *v1.Query) (res []search.Result, err error) {
	graph.Context(ctx, ds.graphProvider, func(inner context.Context) {
		res, err = ds.searcher.Search(inner, q)
	})
	return res, err
}

// Count returns the number of search results from the query
func (ds *searcherImpl) Count(ctx context.Context, q *v1.Query) (int, error) {
	return ds.searcher.Count(ctx, q)
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
func formatSearcher(clusterCVEEdgeIndexer blevesearch.UnsafeSearcher,
	cveIndexer blevesearch.UnsafeSearcher) search.Searcher {
	clusterCVEEdgeSearcher := filtered.UnsafeSearcher(clusterCVEEdgeIndexer, clusterCVEEdgeSAC.GetSACFilter())
	cveSearcher := blevesearch.WrapUnsafeSearcherAsSearcher(cveIndexer)
	compoundSearcher := getCompoundCVESearcher(
		clusterCVEEdgeSearcher,
		cveSearcher,
	)
	return compoundSearcher
}

func getCompoundCVESearcher(
	clusterCVEEdgeSearcher search.Searcher,
	cveSearcher search.Searcher) search.Searcher {
	// The ordering of these is important, so do not change.
	return compound.NewSearcher([]compound.SearcherSpec{
		{
			Searcher:       scoped.WithScoping(cveSearcher, dackbox.ToCategory(v1.SearchCategory_VULNERABILITIES)),
			Transformation: dackbox.GraphTransformations[v1.SearchCategory_VULNERABILITIES][v1.SearchCategory_CLUSTER_VULN_EDGE],
			Options:        cveMappings.OptionsMap,
		},
		{
			IsDefault: true,
			Searcher:  scoped.WithScoping(clusterCVEEdgeSearcher, dackbox.ToCategory(v1.SearchCategory_CLUSTER_VULN_EDGE)),
			Options:   clusterCVEEdgeMappings.OptionsMap,
		},
	})
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
