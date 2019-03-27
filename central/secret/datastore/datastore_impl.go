package datastore

import (
	"github.com/stackrox/rox/central/secret/index"
	"github.com/stackrox/rox/central/secret/search"
	"github.com/stackrox/rox/central/secret/store"
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

func (d *datastoreImpl) ListSecrets() ([]*storage.ListSecret, error) {
	return d.storage.ListAllSecrets()
}

func (d *datastoreImpl) buildIndex() error {
	defer debug.FreeOSMemory()
	secrets, err := d.storage.GetAllSecrets()
	if err != nil {
		return err
	}
	return d.indexer.UpsertSecrets(secrets...)
}

func (d *datastoreImpl) GetSecret(id string) (*storage.Secret, bool, error) {
	return d.storage.GetSecret(id)
}

func (d *datastoreImpl) SearchSecrets(q *v1.Query) ([]*v1.SearchResult, error) {
	return d.searcher.SearchSecrets(q)
}

func (d *datastoreImpl) SearchListSecrets(request *v1.Query) ([]*storage.ListSecret, error) {
	return d.searcher.SearchListSecrets(request)
}

func (d *datastoreImpl) CountSecrets() (int, error) {
	return d.storage.CountSecrets()
}

func (d *datastoreImpl) UpsertSecret(request *storage.Secret) error {
	if err := d.storage.UpsertSecret(request); err != nil {
		return err
	}
	return d.indexer.UpsertSecret(request)
}

func (d *datastoreImpl) RemoveSecret(id string) error {
	if err := d.storage.RemoveSecret(id); err != nil {
		return err
	}
	return d.indexer.RemoveSecret(id)
}

func (d *datastoreImpl) Search(q *v1.Query) ([]searchPkg.Result, error) {
	return d.searcher.Search(q)
}
