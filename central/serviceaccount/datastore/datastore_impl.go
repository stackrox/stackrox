package datastore

import (
	"github.com/stackrox/rox/central/serviceaccount/index"
	"github.com/stackrox/rox/central/serviceaccount/search"
	"github.com/stackrox/rox/central/serviceaccount/store"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	searchPkg "github.com/stackrox/rox/pkg/search"
)

type datastoreImpl struct {
	storage  store.Store
	indexer  index.Indexer
	searcher search.Searcher
}

func (d *datastoreImpl) ListServiceAccounts() ([]*storage.ServiceAccount, error) {
	return d.storage.GetAllServiceAccounts()
}

func (d *datastoreImpl) buildIndex() error {
	serviceAccounts, err := d.storage.GetAllServiceAccounts()
	if err != nil {
		return err
	}
	return d.indexer.UpsertServiceAccounts(serviceAccounts...)
}

func (d *datastoreImpl) GetServiceAccount(id string) (*storage.ServiceAccount, bool, error) {
	return d.storage.GetServiceAccount(id)
}

func (d *datastoreImpl) SearchRawServiceAccounts(q *v1.Query) ([]*storage.ServiceAccount, error) {
	return d.searcher.SearchRawServiceAccounts(q)
}

func (d *datastoreImpl) SearchServiceAccounts(q *v1.Query) ([]*v1.SearchResult, error) {
	return d.searcher.SearchServiceAccounts(q)
}

func (d *datastoreImpl) CountServiceAccounts() (int, error) {
	return d.storage.CountServiceAccounts()
}

func (d *datastoreImpl) UpsertServiceAccount(request *storage.ServiceAccount) error {
	if err := d.storage.UpsertServiceAccount(request); err != nil {
		return err
	}
	return d.indexer.UpsertServiceAccount(request)
}

func (d *datastoreImpl) RemoveServiceAccount(id string) error {
	if err := d.storage.RemoveServiceAccount(id); err != nil {
		return err
	}
	return d.indexer.RemoveServiceAccount(id)
}

func (d *datastoreImpl) Search(q *v1.Query) ([]searchPkg.Result, error) {
	return d.searcher.Search(q)
}
