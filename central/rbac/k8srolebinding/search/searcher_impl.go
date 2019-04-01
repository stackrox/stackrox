package search

import (
	"github.com/blevesearch/bleve"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/rbac/k8srolebinding/search/options"
	"github.com/stackrox/rox/central/rbac/k8srolebinding/store"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/blevesearch"
)

// searcherImpl provides an search implementation for k8s role bindings
type searcherImpl struct {
	storage store.Store
	index   bleve.Index
}

// SearchRoleBindings returns the search results from indexed k8s role bindings for the query.
func (ds *searcherImpl) SearchRoleBindings(q *v1.Query) ([]*v1.SearchResult, error) {
	bindings, results, err := ds.searchRoleBindings(q)
	if err != nil {
		return nil, err
	}
	return convertMany(bindings, results), nil
}

// Search returns the raw search results from the query
func (ds *searcherImpl) Search(q *v1.Query) ([]search.Result, error) {
	return ds.getSearchResults(q)
}

// SearchSecrets returns the secrets and relationships that match the query.
func (ds *searcherImpl) SearchRawRoleBindings(q *v1.Query) ([]*storage.K8SRoleBinding, error) {
	bindings, _, err := ds.searchRoleBindings(q)
	if err != nil {
		return nil, err
	}
	return bindings, nil
}

func (ds *searcherImpl) getSearchResults(q *v1.Query) ([]search.Result, error) {
	results, err := blevesearch.RunSearchRequest(v1.SearchCategory_ROLEBINDINGS, q, ds.index, options.Map)
	if err != nil {
		return nil, errors.Wrapf(err, "error running search request")
	}
	return results, nil
}

func (ds *searcherImpl) searchRoleBindings(q *v1.Query) ([]*storage.K8SRoleBinding, []search.Result, error) {
	results, err := ds.getSearchResults(q)
	if err != nil {
		return nil, nil, err
	}
	var bindings []*storage.K8SRoleBinding
	for _, result := range results {
		binding, exists, err := ds.storage.GetRoleBinding(result.ID)
		if err != nil {
			return nil, nil, err
		}
		// The result may not exist if the object was deleted after the search
		if !exists {
			continue
		}
		bindings = append(bindings, binding)
	}
	return bindings, results, nil
}

func convertMany(bindings []*storage.K8SRoleBinding, results []search.Result) []*v1.SearchResult {
	outputResults := make([]*v1.SearchResult, len(bindings))
	for index, binding := range bindings {
		outputResults[index] = convertOne(binding, &results[index])
	}
	return outputResults
}

func convertOne(binding *storage.K8SRoleBinding, result *search.Result) *v1.SearchResult {
	return &v1.SearchResult{
		Category:       v1.SearchCategory_ROLEBINDINGS,
		Id:             binding.GetId(),
		Name:           binding.GetName(),
		FieldToMatches: search.GetProtoMatchesMap(result.Matches),
		Score:          result.Score,
	}
}
