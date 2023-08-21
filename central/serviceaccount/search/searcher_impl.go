package search

import (
	"context"

	"github.com/stackrox/rox/central/serviceaccount/internal/index"
	"github.com/stackrox/rox/central/serviceaccount/internal/store"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
)

type searcherImpl struct {
	storage store.Store
	indexer index.Indexer
}

// SearchRawServiceAccounts returns the search results from indexed service accounts for the query.
func (ds *searcherImpl) SearchRawServiceAccounts(ctx context.Context, q *v1.Query) ([]*storage.ServiceAccount, error) {
	serviceAccounts, _, err := ds.searchServiceAccounts(ctx, q)
	if err != nil {
		return nil, err
	}

	return serviceAccounts, nil

}

// SearchServiceAccounts returns the search results from indexed service accounts for the query.
func (ds *searcherImpl) SearchServiceAccounts(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error) {
	serviceAccounts, results, err := ds.searchServiceAccounts(ctx, q)
	if err != nil {
		return nil, err
	}

	return convertMany(serviceAccounts, results), nil

}

// Search returns the raw search results from the query
func (ds *searcherImpl) Search(ctx context.Context, q *v1.Query) ([]search.Result, error) {
	return ds.getSearchResults(ctx, q)
}

// Count returns the number of search results from the query
func (ds *searcherImpl) Count(ctx context.Context, q *v1.Query) (int, error) {
	return ds.getCount(ctx, q)
}

func (ds *searcherImpl) getSearchResults(ctx context.Context, q *v1.Query) ([]search.Result, error) {
	return ds.indexer.Search(ctx, q)
}

func (ds *searcherImpl) getCount(ctx context.Context, q *v1.Query) (int, error) {
	return ds.indexer.Count(ctx, q)
}

func (ds *searcherImpl) searchServiceAccounts(ctx context.Context, q *v1.Query) ([]*storage.ServiceAccount, []search.Result, error) {
	results, err := ds.getSearchResults(ctx, q)
	if err != nil {
		return nil, nil, err
	}
	serviceAccounts, missingIndices, err := ds.storage.GetMany(ctx, search.ResultsToIDs(results))
	if err != nil {
		return nil, nil, err
	}
	results = search.RemoveMissingResults(results, missingIndices)
	return serviceAccounts, results, nil
}

func convertMany(serviceAccounts []*storage.ServiceAccount, results []search.Result) []*v1.SearchResult {
	outputResults := make([]*v1.SearchResult, len(serviceAccounts))
	for index, sar := range serviceAccounts {
		outputResults[index] = convertServiceAccount(sar, &results[index])
	}
	return outputResults
}

func convertServiceAccount(sa *storage.ServiceAccount, result *search.Result) *v1.SearchResult {
	return &v1.SearchResult{
		Category:       v1.SearchCategory_SERVICE_ACCOUNTS,
		Id:             sa.GetId(),
		Name:           sa.GetName(),
		FieldToMatches: search.GetProtoMatchesMap(result.Matches),
		Score:          result.Score,
	}
}
