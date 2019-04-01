package datastore

import (
	"github.com/stackrox/rox/central/rbac/k8srolebinding/index"
	"github.com/stackrox/rox/central/rbac/k8srolebinding/search"
	"github.com/stackrox/rox/central/rbac/k8srolebinding/store"
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

func (d *datastoreImpl) ListRoleBindings() ([]*storage.K8SRoleBinding, error) {
	return d.storage.ListAllRoleBindings()
}

func (d *datastoreImpl) buildIndex() error {
	defer debug.FreeOSMemory()
	bindings, err := d.storage.ListAllRoleBindings()
	if err != nil {
		return err
	}
	return d.indexer.UpsertRoleBindings(bindings...)
}

func (d *datastoreImpl) GetRoleBinding(id string) (*storage.K8SRoleBinding, bool, error) {
	return d.storage.GetRoleBinding(id)
}

func (d *datastoreImpl) SearchRoleBindings(q *v1.Query) ([]*v1.SearchResult, error) {
	return d.searcher.SearchRoleBindings(q)
}

func (d *datastoreImpl) SearchRawRoleBindings(request *v1.Query) ([]*storage.K8SRoleBinding, error) {
	return d.searcher.SearchRawRoleBindings(request)
}

func (d *datastoreImpl) CountRoleBindings() (int, error) {
	return d.storage.CountRoleBindings()
}

func (d *datastoreImpl) UpsertRoleBinding(request *storage.K8SRoleBinding) error {
	if err := d.storage.UpsertRoleBinding(request); err != nil {
		return err
	}
	return d.indexer.UpsertRoleBinding(request)
}

func (d *datastoreImpl) RemoveRoleBinding(id string) error {
	if err := d.storage.RemoveRoleBinding(id); err != nil {
		return err
	}
	return d.indexer.RemoveRoleBinding(id)
}

func (d *datastoreImpl) Search(q *v1.Query) ([]searchPkg.Result, error) {
	return d.searcher.Search(q)
}
