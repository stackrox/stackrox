package search

import (
	"fmt"

	"github.com/blevesearch/bleve"
	"github.com/stackrox/rox/central/serviceaccount/search/options"
	"github.com/stackrox/rox/central/serviceaccount/store"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/blevesearch"
)

type searcherImpl struct {
	storage store.Store
	index   bleve.Index
}

// SearchRawServiceAccounts returns the search results from indexed service accounts for the query.
func (ds *searcherImpl) SearchRawServiceAccounts(q *v1.Query) ([]*storage.ServiceAccount, error) {
	serviceAccounts, _, err := ds.searchServiceAccounts(q)
	if err != nil {
		return nil, err
	}

	return serviceAccounts, nil

}

// SearchServiceAccounts returns the search results from indexed service accounts for the query.
func (ds *searcherImpl) SearchServiceAccounts(q *v1.Query) ([]*v1.SearchResult, error) {
	serviceAccounts, results, err := ds.searchServiceAccounts(q)
	if err != nil {
		return nil, err
	}

	return convertMany(serviceAccounts, results), nil

}

// Search returns the raw search results from the query
func (ds *searcherImpl) Search(q *v1.Query) ([]search.Result, error) {
	return ds.getSearchResults(q)
}

func (ds *searcherImpl) getSearchResults(q *v1.Query) ([]search.Result, error) {
	results, err := blevesearch.RunSearchRequest(v1.SearchCategory_SERVICE_ACCOUNTS, q, ds.index, options.Map)
	if err != nil {
		return nil, fmt.Errorf("running search request: %v", err)
	}
	return results, nil
}

func (ds *searcherImpl) searchServiceAccounts(q *v1.Query) ([]*storage.ServiceAccount, []search.Result, error) {
	results, err := ds.getSearchResults(q)
	if err != nil {
		return nil, nil, err
	}
	var serviceAccounts []*storage.ServiceAccount
	for _, result := range results {
		sa, exists, err := ds.storage.GetServiceAccount(result.ID)
		if err != nil {
			return nil, nil, err
		}
		// The result may not exist if the object was deleted after the search
		if !exists {
			continue
		}
		serviceAccounts = append(serviceAccounts, sa)
	}
	return serviceAccounts, results, nil
}

func convertMany(serviceAccounts []*storage.ServiceAccount, results []search.Result) []*v1.SearchResult {
	outputResults := make([]*v1.SearchResult, len(serviceAccounts), len(serviceAccounts))
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
