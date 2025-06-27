package datastore

import (
	"context"

	"github.com/stackrox/rox/central/rbac/k8srole/internal/store"
	"github.com/stackrox/rox/central/rbac/k8srole/search"
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
	storage  store.Store
	searcher search.Searcher
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
	return d.searcher.SearchRoles(ctx, q)
}

func (d *datastoreImpl) SearchRawRoles(ctx context.Context, request *v1.Query) ([]*storage.K8SRole, error) {
	return d.searcher.SearchRawRoles(ctx, request)
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
	return d.searcher.Search(ctx, q)
}

// Count returns the number of search results from the query
func (d *datastoreImpl) Count(ctx context.Context, q *v1.Query) (int, error) {
	return d.searcher.Count(ctx, q)
}
