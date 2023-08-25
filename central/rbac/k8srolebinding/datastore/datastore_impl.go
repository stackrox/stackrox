package datastore

import (
	"context"
	"time"

	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/central/rbac/k8srolebinding/internal/store"
	"github.com/stackrox/rox/central/rbac/k8srolebinding/search"
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
	storage  store.Store
	searcher search.Searcher
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
	return d.searcher.SearchRoleBindings(ctx, q)
}

func (d *datastoreImpl) SearchRawRoleBindings(ctx context.Context, request *v1.Query) ([]*storage.K8SRoleBinding, error) {
	return d.searcher.SearchRawRoleBindings(ctx, request)
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
	return d.searcher.Search(ctx, q)
}

func (d *datastoreImpl) Count(ctx context.Context, q *v1.Query) (int, error) {
	return d.searcher.Count(ctx, q)
}

func (d *datastoreImpl) GetManyRoleBindings(ctx context.Context, ids []string) ([]*storage.K8SRoleBinding, []int, error) {
	return d.storage.GetMany(ctx, ids)
}
