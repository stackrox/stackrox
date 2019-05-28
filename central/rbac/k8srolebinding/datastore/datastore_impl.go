package datastore

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/rbac/k8srolebinding/internal/index"
	"github.com/stackrox/rox/central/rbac/k8srolebinding/internal/store"
	"github.com/stackrox/rox/central/rbac/k8srolebinding/search"
	"github.com/stackrox/rox/central/role/resources"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/debug"
	"github.com/stackrox/rox/pkg/sac"
	searchPkg "github.com/stackrox/rox/pkg/search"
)

var (
	k8sRoleBindingsSAC = sac.ForResource(resources.K8sRoleBinding)
)

type datastoreImpl struct {
	storage  store.Store
	indexer  index.Indexer
	searcher search.Searcher
}

func (d *datastoreImpl) ListRoleBindings(ctx context.Context) ([]*storage.K8SRoleBinding, error) {
	if ok, err := k8sRoleBindingsSAC.ReadAllowed(ctx); err != nil {
		return nil, err
	} else if ok {
		return d.storage.ListRoleBindings()
	}

	return d.SearchRawRoleBindings(ctx, searchPkg.EmptyQuery())
}

func (d *datastoreImpl) buildIndex() error {
	defer debug.FreeOSMemory()
	bindings, err := d.storage.ListRoleBindings()
	if err != nil {
		return err
	}
	return d.indexer.AddK8sRoleBindings(bindings)
}

func (d *datastoreImpl) GetRoleBinding(ctx context.Context, id string) (*storage.K8SRoleBinding, bool, error) {
	binding, found, err := d.storage.GetRoleBinding(id)
	if err != nil || !found {
		return nil, false, err
	}

	if ok, err := k8sRoleBindingsSAC.ScopeChecker(ctx, storage.Access_READ_ACCESS).ForNamespaceScopedObject(binding).Allowed(ctx); err != nil || !ok {
		return nil, false, err
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
		return errors.New("permission denied")
	}

	if err := d.storage.UpsertRoleBinding(request); err != nil {
		return err
	}
	return d.indexer.AddK8sRoleBinding(request)
}

func (d *datastoreImpl) RemoveRoleBinding(ctx context.Context, id string) error {
	if ok, err := k8sRoleBindingsSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return errors.New("permission denied")
	}

	if err := d.storage.DeleteRoleBinding(id); err != nil {
		return err
	}
	return d.indexer.DeleteK8sRoleBinding(id)
}

func (d *datastoreImpl) Search(ctx context.Context, q *v1.Query) ([]searchPkg.Result, error) {
	return d.searcher.Search(ctx, q)
}
