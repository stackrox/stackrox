package datastore

import (
	"context"
	"strings"

	"github.com/stackrox/rox/central/rbac/k8srole/internal/store"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	searchPkg "github.com/stackrox/rox/pkg/search"
)

type datastoreImpl struct {
	storage store.Store
}

func (d *datastoreImpl) GetRole(ctx context.Context, id string) (*storage.K8SRole, bool, error) {
	role, found, err := d.storage.Get(ctx, id)
	if err != nil || !found {
		return nil, false, err
	}

	return role, true, nil
}

func (d *datastoreImpl) SearchRoles(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error) {
	if q == nil {
		q = searchPkg.EmptyQuery()
	}
	qClone := q.CloneVT()

	// Add name field to select columns
	qClone.Selects = append(qClone.GetSelects(), searchPkg.NewQuerySelect(searchPkg.RoleName).Proto())

	results, err := d.Search(ctx, qClone)
	if err != nil {
		return nil, err
	}

	// Extract name from FieldValues and populate Name in search results
	searchTag := strings.ToLower(searchPkg.RoleName.String())
	for i := range results {
		if results[i].FieldValues != nil {
			if nameVal, ok := results[i].FieldValues[searchTag]; ok {
				results[i].Name = nameVal
			}
		}
	}

	return searchPkg.ResultsToSearchResultProtos(results, &K8SRoleSearchResultConverter{}), nil
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
	return d.storage.Upsert(ctx, request)
}

func (d *datastoreImpl) RemoveRole(ctx context.Context, id string) error {
	return d.storage.Delete(ctx, id)
}

func (d *datastoreImpl) Search(ctx context.Context, q *v1.Query) ([]searchPkg.Result, error) {
	return d.storage.Search(ctx, q)
}

// Count returns the number of search results from the query
func (d *datastoreImpl) Count(ctx context.Context, q *v1.Query) (int, error) {
	return d.storage.Count(ctx, q)
}

type K8SRoleSearchResultConverter struct{}

func (c *K8SRoleSearchResultConverter) BuildName(result *searchPkg.Result) string {
	return result.Name
}

func (c *K8SRoleSearchResultConverter) BuildLocation(result *searchPkg.Result) string {
	// K8SRole does not have a location
	return ""
}

func (c *K8SRoleSearchResultConverter) GetCategory() v1.SearchCategory {
	return v1.SearchCategory_ROLES
}

func (c *K8SRoleSearchResultConverter) GetScore(result *searchPkg.Result) float64 {
	return result.Score
}
