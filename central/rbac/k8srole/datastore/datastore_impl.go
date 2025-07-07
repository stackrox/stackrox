package datastore

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/rbac/k8srole/internal/store"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	searchPkg "github.com/stackrox/rox/pkg/search"
)

var (
	k8sRolesSAC = sac.ForResource(resources.K8sRole)
)

type datastoreImpl struct {
	storage store.Store
}

func (d *datastoreImpl) GetRole(ctx context.Context, id string) (*storage.K8SRole, bool, error) {
	role, found, err := d.storage.Get(ctx, id)
	if err != nil || !found {
		return nil, false, err
	}

	if !k8sRolesSAC.ScopeChecker(ctx, storage.Access_READ_ACCESS).ForNamespaceScopedObject(role).IsAllowed() {
		return nil, false, nil
	}
	return role, true, nil
}

func (d *datastoreImpl) SearchRoles(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error) {
	results, err := d.Search(ctx, q)
	if err != nil {
		return nil, err
	}
	roles, missingIndices, err := d.storage.GetMany(ctx, searchPkg.ResultsToIDs(results))
	if err != nil {
		return nil, err
	}
	results = searchPkg.RemoveMissingResults(results, missingIndices)

	return convertMany(roles, results)
}

func (d *datastoreImpl) SearchRawRoles(ctx context.Context, request *v1.Query) ([]*storage.K8SRole, error) {
	roles := make([]*storage.K8SRole, 0)
	err := d.storage.GetByQueryFn(ctx, request, func(role *storage.K8SRole) error {
		roles = append(roles, role)
		return nil
	})
	return roles, err
}

func (d *datastoreImpl) UpsertRole(ctx context.Context, request *storage.K8SRole) error {
	if ok, err := k8sRolesSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}

	return d.storage.Upsert(ctx, request)
}

func (d *datastoreImpl) RemoveRole(ctx context.Context, id string) error {
	if ok, err := k8sRolesSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}

	return d.storage.Delete(ctx, id)
}

func (d *datastoreImpl) Search(ctx context.Context, q *v1.Query) ([]searchPkg.Result, error) {
	return d.storage.Search(ctx, q)
}

// Count returns the number of search results from the query
func (d *datastoreImpl) Count(ctx context.Context, q *v1.Query) (int, error) {
	return d.storage.Count(ctx, q)
}

func convertMany(roles []*storage.K8SRole, results []searchPkg.Result) ([]*v1.SearchResult, error) {
	if len(roles) != len(results) {
		return nil, errors.New("mismatch between search results and retrieved roles")
	}

	outputResults := make([]*v1.SearchResult, len(roles))
	for index, role := range roles {
		outputResults[index] = convertOne(role, &results[index])
	}
	return outputResults, nil
}

func convertOne(role *storage.K8SRole, result *searchPkg.Result) *v1.SearchResult {
	return &v1.SearchResult{
		Category:       v1.SearchCategory_ROLES,
		Id:             role.GetId(),
		Name:           role.GetName(),
		FieldToMatches: searchPkg.GetProtoMatchesMap(result.Matches),
		Score:          result.Score,
	}
}
