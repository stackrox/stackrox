package datastore

import (
	"context"

	"github.com/stackrox/rox/central/rbac/k8srolebinding/internal/index"
	"github.com/stackrox/rox/central/rbac/k8srolebinding/internal/store"
	"github.com/stackrox/rox/central/rbac/k8srolebinding/search"
	"github.com/stackrox/rox/central/role/resources"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/debug"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/sac"
	searchPkg "github.com/stackrox/rox/pkg/search"
)

const (
	batchSize = 1000
)

var (
	k8sRoleBindingsSAC = sac.ForResource(resources.K8sRoleBinding)
)

type datastoreImpl struct {
	storage  store.Store
	indexer  index.Indexer
	searcher search.Searcher
}

func (d *datastoreImpl) buildIndex(ctx context.Context) error {
	if features.PostgresDatastore.Enabled() {
		return nil
	}
	defer debug.FreeOSMemory()
	log.Info("[STARTUP] Indexing rolebindings")

	var bindings []*storage.K8SRoleBinding
	var count int
	err := d.storage.Walk(ctx, func(binding *storage.K8SRoleBinding) error {
		bindings = append(bindings, binding)
		if len(bindings) == batchSize {
			if err := d.indexer.AddK8SRoleBindings(bindings); err != nil {
				return err
			}
			bindings = bindings[:0]
		}
		count++
		return nil
	})
	if err != nil {
		return err
	}
	if err := d.indexer.AddK8SRoleBindings(bindings); err != nil {
		return err
	}
	log.Infof("[STARTUP] Successfully indexed %d rolebindings", count)
	return nil
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

	if err := d.storage.Upsert(ctx, request); err != nil {
		return err
	}
	return d.indexer.AddK8SRoleBinding(request)
}

func (d *datastoreImpl) RemoveRoleBinding(ctx context.Context, id string) error {
	if ok, err := k8sRoleBindingsSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}

	if err := d.storage.Delete(ctx, id); err != nil {
		return err
	}
	return d.indexer.DeleteK8SRoleBinding(id)
}

func (d *datastoreImpl) Search(ctx context.Context, q *v1.Query) ([]searchPkg.Result, error) {
	return d.searcher.Search(ctx, q)
}

func (d *datastoreImpl) Count(ctx context.Context, q *v1.Query) (int, error) {
	return d.searcher.Count(ctx, q)
}
