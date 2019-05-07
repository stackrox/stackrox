package datastore

import (
	"context"

	"github.com/stackrox/rox/central/rbac/k8srole/index"
	"github.com/stackrox/rox/central/rbac/k8srole/search"
	"github.com/stackrox/rox/central/rbac/k8srole/store"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/debug"
	searchPkg "github.com/stackrox/rox/pkg/search"
)

type datastoreImpl struct {
	storage  store.Store
	indexer  index.Indexer
	searcher search.Searcher
}

func (d *datastoreImpl) ListRoles(ctx context.Context) ([]*storage.K8SRole, error) {
	return d.storage.ListAllRoles()
}

func (d *datastoreImpl) buildIndex() error {
	defer debug.FreeOSMemory()
	roles, err := d.storage.ListAllRoles()
	if err != nil {
		return err
	}
	return d.indexer.UpsertRoles(roles...)
}

func (d *datastoreImpl) GetRole(ctx context.Context, id string) (*storage.K8SRole, bool, error) {
	return d.storage.GetRole(id)
}

func (d *datastoreImpl) SearchRoles(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error) {
	return d.searcher.SearchRoles(q)
}

func (d *datastoreImpl) SearchRawRoles(ctx context.Context, request *v1.Query) ([]*storage.K8SRole, error) {
	return d.searcher.SearchRawRoles(request)
}

func (d *datastoreImpl) CountRoles(ctx context.Context) (int, error) {
	return d.storage.CountRoles()
}

func (d *datastoreImpl) UpsertRole(ctx context.Context, request *storage.K8SRole) error {
	if err := d.storage.UpsertRole(request); err != nil {
		return err
	}
	return d.indexer.UpsertRole(request)
}

func (d *datastoreImpl) RemoveRole(ctx context.Context, id string) error {
	if err := d.storage.RemoveRole(id); err != nil {
		return err
	}
	return d.indexer.RemoveRole(id)
}

func (d *datastoreImpl) Search(ctx context.Context, q *v1.Query) ([]searchPkg.Result, error) {
	return d.searcher.Search(q)
}
