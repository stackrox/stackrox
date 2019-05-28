package datastore

import (
	"context"
	"errors"

	"github.com/stackrox/rox/central/rbac/k8srole/internal/index"
	"github.com/stackrox/rox/central/rbac/k8srole/internal/store"
	"github.com/stackrox/rox/central/rbac/k8srole/search"
	"github.com/stackrox/rox/central/role/resources"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/debug"
	"github.com/stackrox/rox/pkg/sac"
	searchPkg "github.com/stackrox/rox/pkg/search"
)

var (
	k8sRolesSAC = sac.ForResource(resources.K8sRole)
)

type datastoreImpl struct {
	storage  store.Store
	indexer  index.Indexer
	searcher search.Searcher
}

func (d *datastoreImpl) ListRoles(ctx context.Context) ([]*storage.K8SRole, error) {
	if ok, err := k8sRolesSAC.ReadAllowed(ctx); err != nil {
		return nil, err
	} else if ok {
		return d.storage.ListRoles()
	}

	return d.SearchRawRoles(ctx, searchPkg.EmptyQuery())
}

func (d *datastoreImpl) buildIndex() error {
	defer debug.FreeOSMemory()
	roles, err := d.storage.ListRoles()
	if err != nil {
		return err
	}
	return d.indexer.AddK8SRoles(roles)
}

func (d *datastoreImpl) GetRole(ctx context.Context, id string) (*storage.K8SRole, bool, error) {
	role, found, err := d.storage.GetRole(id)
	if err != nil || !found {
		return nil, false, err
	}

	if ok, err := k8sRolesSAC.ScopeChecker(ctx, storage.Access_READ_ACCESS).ForNamespaceScopedObject(role).Allowed(ctx); err != nil || !ok {
		return nil, false, err
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
		return errors.New("permission denied")
	}

	if err := d.storage.UpsertRole(request); err != nil {
		return err
	}
	return d.indexer.AddK8SRole(request)
}

func (d *datastoreImpl) RemoveRole(ctx context.Context, id string) error {
	if ok, err := k8sRolesSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return errors.New("permission denied")
	}

	if err := d.storage.DeleteRole(id); err != nil {
		return err
	}
	return d.indexer.DeleteK8SRole(id)
}

func (d *datastoreImpl) Search(ctx context.Context, q *v1.Query) ([]searchPkg.Result, error) {
	return d.searcher.Search(ctx, q)
}
