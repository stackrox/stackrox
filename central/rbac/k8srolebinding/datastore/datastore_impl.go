package datastore

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/central/rbac/k8srolebinding/internal/store"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	searchPkg "github.com/stackrox/rox/pkg/search"
)

var (
	k8sRoleBindingsSAC = sac.ForResource(resources.K8sRoleBinding)
)

type datastoreImpl struct {
	storage store.Store
}

func (d *datastoreImpl) GetRoleBinding(ctx context.Context, id string) (*storage.K8SRoleBinding, bool, error) {
	binding, found, err := d.storage.Get(ctx, id)
	if err != nil || !found {
		return nil, false, err
	}

	if !k8sRoleBindingsSAC.ScopeChecker(ctx, storage.Access_READ_ACCESS).ForNamespaceScopedObject(binding).IsAllowed() {
		return nil, false, nil
	}

	return binding, true, nil
}

func (d *datastoreImpl) SearchRoleBindings(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error) {
	results, err := d.Search(ctx, q)
	if err != nil {
		return nil, err
	}
	bindings, missingIndices, err := d.storage.GetMany(ctx, searchPkg.ResultsToIDs(results))
	if err != nil {
		return nil, err
	}
	results = searchPkg.RemoveMissingResults(results, missingIndices)
	return convertMany(bindings, results)
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
	if ok, err := k8sRoleBindingsSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}

	return d.storage.Upsert(ctx, request)
}

func (d *datastoreImpl) RemoveRoleBinding(ctx context.Context, id string) error {
	if ok, err := k8sRoleBindingsSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}

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

func convertMany(bindings []*storage.K8SRoleBinding, results []searchPkg.Result) ([]*v1.SearchResult, error) {
	if len(bindings) != len(results) {
		return nil, errors.New("mismatch between search results and retrieved role bindings")
	}

	outputResults := make([]*v1.SearchResult, len(bindings))
	for index, binding := range bindings {
		outputResults[index] = convertOne(binding, &results[index])
	}
	return outputResults, nil
}

func convertOne(binding *storage.K8SRoleBinding, result *searchPkg.Result) *v1.SearchResult {
	sr := &v1.SearchResult{}
	sr.SetCategory(v1.SearchCategory_ROLEBINDINGS)
	sr.SetId(binding.GetId())
	sr.SetName(binding.GetName())
	sr.SetFieldToMatches(searchPkg.GetProtoMatchesMap(result.Matches))
	sr.SetScore(result.Score)
	return sr
}
