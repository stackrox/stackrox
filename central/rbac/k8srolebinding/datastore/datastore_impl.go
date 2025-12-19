package datastore

import (
	"context"
	"strings"
	"time"

	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/central/rbac/k8srolebinding/internal/store"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	searchPkg "github.com/stackrox/rox/pkg/search"
)

type datastoreImpl struct {
	storage store.Store
}

func (d *datastoreImpl) GetRoleBinding(ctx context.Context, id string) (*storage.K8SRoleBinding, bool, error) {
	binding, found, err := d.storage.Get(ctx, id)
	if err != nil || !found {
		return nil, false, err
	}

	return binding, true, nil
}

func (d *datastoreImpl) SearchRoleBindings(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error) {
	if q == nil {
		q = searchPkg.EmptyQuery()
	}
	qClone := q.CloneVT()

	// Add name field to select columns
	qClone.Selects = append(qClone.GetSelects(), searchPkg.NewQuerySelect(searchPkg.RoleBindingName).Proto())

	results, err := d.Search(ctx, qClone)
	if err != nil {
		return nil, err
	}

	// Extract name from FieldValues and populate Name in search results
	searchTag := strings.ToLower(searchPkg.RoleBindingName.String())
	for i := range results {
		if results[i].FieldValues != nil {
			if nameVal, ok := results[i].FieldValues[searchTag]; ok {
				results[i].Name = nameVal
			}
		}
	}

	return searchPkg.ResultsToSearchResultProtos(results, &K8SRoleBindingSearchResultConverter{}), nil
}

func (d *datastoreImpl) SearchRawRoleBindings(ctx context.Context, request *v1.Query) ([]*storage.K8SRoleBinding, error) {
	bindings := make([]*storage.K8SRoleBinding, 0)
	err := d.storage.GetByQueryFn(ctx, request, func(roleBinding *storage.K8SRoleBinding) error {
		bindings = append(bindings, roleBinding)
		return nil
	})
	return bindings, err
}

func (d *datastoreImpl) UpsertRoleBinding(ctx context.Context, request *storage.K8SRoleBinding) error {
	return d.storage.Upsert(ctx, request)
}

func (d *datastoreImpl) RemoveRoleBinding(ctx context.Context, id string) error {
	return d.storage.Delete(ctx, id)
}

func (d *datastoreImpl) Search(ctx context.Context, q *v1.Query) ([]searchPkg.Result, error) {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), "K8SRoleBinding", "Search")
	return d.storage.Search(ctx, q)
}

func (d *datastoreImpl) Count(ctx context.Context, q *v1.Query) (int, error) {
	return d.storage.Count(ctx, q)
}

func (d *datastoreImpl) GetManyRoleBindings(ctx context.Context, ids []string) ([]*storage.K8SRoleBinding, []int, error) {
	return d.storage.GetMany(ctx, ids)
}

type K8SRoleBindingSearchResultConverter struct{}

func (c *K8SRoleBindingSearchResultConverter) BuildName(result *searchPkg.Result) string {
	return result.Name
}

func (c *K8SRoleBindingSearchResultConverter) BuildLocation(result *searchPkg.Result) string {
	// K8SRoleBinding does not have a location
	return ""
}

func (c *K8SRoleBindingSearchResultConverter) GetCategory() v1.SearchCategory {
	return v1.SearchCategory_ROLEBINDINGS
}

func (c *K8SRoleBindingSearchResultConverter) GetScore(result *searchPkg.Result) float64 {
	return result.Score
}
