package search

import (
	"context"

	"github.com/stackrox/rox/central/cve/node/datastore/store"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
)

type searcherImpl struct {
	storage  store.Store
	searcher search.Searcher
}

func (ds *searcherImpl) SearchCVEs(ctx context.Context, q *v1.Query, allowOrphaned bool) ([]*v1.SearchResult, error) {
	results, err := ds.getSearchResults(ctx, q, allowOrphaned)
	if err != nil {
		return nil, err
	}
	return ds.resultsToSearchResults(ctx, results)
}

func (ds *searcherImpl) Search(ctx context.Context, q *v1.Query, allowOrphaned bool) ([]search.Result, error) {
	return ds.getSearchResults(ctx, q, allowOrphaned)
}

// Count returns the number of search results from the query
func (ds *searcherImpl) Count(ctx context.Context, q *v1.Query, allowOrphaned bool) (int, error) {
	return ds.getCount(ctx, q, allowOrphaned)
}

func (ds *searcherImpl) SearchRawCVEs(ctx context.Context, q *v1.Query, allowOrphaned bool) ([]*storage.NodeCVE, error) {
	return ds.searchCVEs(ctx, q, allowOrphaned)
}

func (ds *searcherImpl) getSearchResults(ctx context.Context, q *v1.Query, allowOrphaned bool) (res []search.Result, err error) {
	if allowOrphaned {
		return ds.searcher.Search(ctx, q)
	}
	return ds.searcher.Search(ctx, withoutOrphanedCVEsQuery(q))
}

func (ds *searcherImpl) getCount(ctx context.Context, q *v1.Query, allowOrphaned bool) (count int, err error) {
	if allowOrphaned {
		return ds.searcher.Count(ctx, q)
	}
	return ds.searcher.Count(ctx, withoutOrphanedCVEsQuery(q))
}

func (ds *searcherImpl) resultsToCVEs(ctx context.Context, results []search.Result) ([]*storage.NodeCVE, []int, error) {
	return ds.storage.GetMany(ctx, search.ResultsToIDs(results))
}

func (ds *searcherImpl) resultsToSearchResults(ctx context.Context, results []search.Result) ([]*v1.SearchResult, error) {
	cves, missingIndices, err := ds.resultsToCVEs(ctx, results)
	if err != nil {
		return nil, err
	}
	results = search.RemoveMissingResults(results, missingIndices)
	return convertMany(cves, results), nil
}

func convertMany(cves []*storage.NodeCVE, results []search.Result) []*v1.SearchResult {
	outputResults := make([]*v1.SearchResult, len(cves))
	for index, sar := range cves {
		outputResults[index] = convertOne(sar, &results[index])
	}
	return outputResults
}

func convertOne(cve *storage.NodeCVE, result *search.Result) *v1.SearchResult {
	return &v1.SearchResult{
		Category:       v1.SearchCategory_NODE_VULNERABILITIES,
		Id:             cve.GetId(),
		Name:           cve.GetCveBaseInfo().GetCve(),
		FieldToMatches: search.GetProtoMatchesMap(result.Matches),
		Score:          result.Score,
	}
}

func (ds *searcherImpl) searchCVEs(ctx context.Context, q *v1.Query, allowOrphaned bool) ([]*storage.NodeCVE, error) {
	results, err := ds.Search(ctx, q, allowOrphaned)
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

func withoutOrphanedCVEsQuery(q *v1.Query) *v1.Query {
	ret := search.ConjunctionQuery(q, search.NewQueryBuilder().AddBools(search.CVEOrphaned, false).ProtoQuery())
	ret.Pagination = q.GetPagination()
	return ret
}
