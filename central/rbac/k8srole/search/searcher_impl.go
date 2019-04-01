package search

import (
	"github.com/blevesearch/bleve"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/rbac/k8srole/search/options"
	"github.com/stackrox/rox/central/rbac/k8srole/store"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/blevesearch"
)

// searcherImpl provides an intermediary implementation layer for AlertStorage.
type searcherImpl struct {
	storage store.Store
	index   bleve.Index
}

// SearchSecrets returns the search results from indexed k8s roles for the query.
func (ds *searcherImpl) SearchRoles(q *v1.Query) ([]*v1.SearchResult, error) {
	roles, results, err := ds.searchRoles(q)
	if err != nil {
		return nil, err
	}

	return convertMany(roles, results), nil
}

// Search returns the raw search results from the query
func (ds *searcherImpl) Search(q *v1.Query) ([]search.Result, error) {
	return ds.getSearchResults(q)
}

// SearchSecrets returns the secrets and relationships that match the query.
func (ds *searcherImpl) SearchRawRoles(q *v1.Query) ([]*storage.K8SRole, error) {
	roles, _, err := ds.searchRoles(q)
	if err != nil {
		return nil, err
	}
	return roles, nil
}

func (ds *searcherImpl) searchRoles(q *v1.Query) ([]*storage.K8SRole, []search.Result, error) {
	results, err := ds.getSearchResults(q)
	if err != nil {
		return nil, nil, err
	}
	var roles []*storage.K8SRole
	for _, result := range results {
		role, exists, err := ds.storage.GetRole(result.ID)
		if err != nil {
			return nil, nil, err
		}
		// The result may not exist if the object was deleted after the search
		if !exists {
			continue
		}
		roles = append(roles, role)
	}
	return roles, results, nil
}

func (ds *searcherImpl) getSearchResults(q *v1.Query) ([]search.Result, error) {
	results, err := blevesearch.RunSearchRequest(v1.SearchCategory_ROLES, q, ds.index, options.Map)
	if err != nil {
		return nil, errors.Wrapf(err, "error running search request")
	}
	return results, nil
}

func convertMany(roles []*storage.K8SRole, results []search.Result) []*v1.SearchResult {
	outputResults := make([]*v1.SearchResult, len(roles))
	for index, role := range roles {
		outputResults[index] = convertOne(role, &results[index])
	}
	return outputResults
}

func convertOne(role *storage.K8SRole, result *search.Result) *v1.SearchResult {
	return &v1.SearchResult{
		Category:       v1.SearchCategory_ROLES,
		Id:             role.GetId(),
		Name:           role.GetName(),
		FieldToMatches: search.GetProtoMatchesMap(result.Matches),
		Score:          result.Score,
	}
}
