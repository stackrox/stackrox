package search

import (
	"context"

	"github.com/stackrox/rox/central/rbac/k8srole/internal/index"
	"github.com/stackrox/rox/central/rbac/k8srole/internal/store"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
)

var (
	k8sRolesSACPgSearchHelper = sac.ForResource(resources.K8sRole).MustCreatePgSearchHelper()
)

// searcherImpl provides an intermediary implementation layer for AlertStorage.
type searcherImpl struct {
	storage store.Store
	indexer index.Indexer
}

// SearchRoles returns the search results from indexed k8s roles for the query.
func (ds *searcherImpl) SearchRoles(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error) {
	roles, results, err := ds.searchRoles(ctx, q)
	if err != nil {
		return nil, err
	}

	return convertMany(roles, results), nil
}

// Search returns the raw search results from the query.
func (ds *searcherImpl) Search(ctx context.Context, q *v1.Query) ([]search.Result, error) {
	return ds.getSearchResults(ctx, q)
}

// Count returns the number of search results from the query.
func (ds *searcherImpl) Count(ctx context.Context, q *v1.Query) (int, error) {
	return ds.getCountResults(ctx, q)
}

// SearchRawRoles returns the roles and relationships that match the query.
func (ds *searcherImpl) SearchRawRoles(ctx context.Context, q *v1.Query) ([]*storage.K8SRole, error) {
	roles, _, err := ds.searchRoles(ctx, q)
	if err != nil {
		return nil, err
	}
	return roles, nil
}

func (ds *searcherImpl) searchRoles(ctx context.Context, q *v1.Query) ([]*storage.K8SRole, []search.Result, error) {
	results, err := ds.getSearchResults(ctx, q)
	if err != nil {
		return nil, nil, err
	}
	roles, missingIndices, err := ds.storage.GetMany(ctx, search.ResultsToIDs(results))
	if err != nil {
		return nil, nil, err
	}
	results = search.RemoveMissingResults(results, missingIndices)
	return roles, results, nil
}

func (ds *searcherImpl) getSearchResults(ctx context.Context, q *v1.Query) ([]search.Result, error) {
	return k8sRolesSACPgSearchHelper.FilteredSearcher(ds.indexer).Search(ctx, q)
}

func (ds *searcherImpl) getCountResults(ctx context.Context, q *v1.Query) (int, error) {
	return k8sRolesSACPgSearchHelper.FilteredSearcher(ds.indexer).Count(ctx, q)
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
