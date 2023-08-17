package search

import (
	"context"

	"github.com/stackrox/rox/central/rbac/k8srolebinding/internal/index"
	"github.com/stackrox/rox/central/rbac/k8srolebinding/internal/store"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
)

var (
	k8sRoleBindingsSACPostgresSearchHelper = sac.ForResource(resources.K8sRoleBinding).
		MustCreatePgSearchHelper()
)

// searcherImpl provides a search implementation for k8s role bindings.
type searcherImpl struct {
	storage store.Store
	index   index.Indexer
}

// SearchRoleBindings returns the search results from indexed k8s role bindings for the query.
func (ds *searcherImpl) SearchRoleBindings(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error) {
	bindings, results, err := ds.searchRoleBindings(ctx, q)
	if err != nil {
		return nil, err
	}
	return convertMany(bindings, results), nil
}

// Search returns the raw search results from the query.
func (ds *searcherImpl) Search(ctx context.Context, q *v1.Query) ([]search.Result, error) {
	// return k8sRoleBindingsSACPostgresSearchHelper.FilteredSearcher(ds.index).Search(ctx, q)
	return ds.index.Search(ctx, q)
}

// Count returns the number of search results from the query.
func (ds *searcherImpl) Count(ctx context.Context, q *v1.Query) (int, error) {
	// return k8sRoleBindingsSACPostgresSearchHelper.FilteredSearcher(ds.index).Count(ctx, q)
	return ds.index.Count(ctx, q)
}

// SearchRawRoleBindings returns the rolebindings that match the query.
func (ds *searcherImpl) SearchRawRoleBindings(ctx context.Context, q *v1.Query) ([]*storage.K8SRoleBinding, error) {
	bindings, _, err := ds.searchRoleBindings(ctx, q)
	if err != nil {
		return nil, err
	}
	return bindings, nil
}

func (ds *searcherImpl) searchRoleBindings(ctx context.Context, q *v1.Query) ([]*storage.K8SRoleBinding, []search.Result, error) {
	results, err := ds.Search(ctx, q)
	if err != nil {
		return nil, nil, err
	}
	bindings, missingIndices, err := ds.storage.GetMany(ctx, search.ResultsToIDs(results))
	if err != nil {
		return nil, nil, err
	}
	results = search.RemoveMissingResults(results, missingIndices)
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
