package datastore

import (
	"github.com/stackrox/rox/central/secret/index"
	"github.com/stackrox/rox/central/secret/search"
	"github.com/stackrox/rox/central/secret/store"
	"github.com/stackrox/rox/generated/api/v1"
)

type datastoreImpl struct {
	storage  store.Store
	indexer  index.Indexer
	searcher search.Searcher
}

func (d *datastoreImpl) GetSecret(id string) (*v1.Secret, bool, error) {
	return d.storage.GetSecret(id)
}

func (d *datastoreImpl) SearchSecrets(q *v1.Query) ([]*v1.SearchResult, error) {
	return d.searcher.SearchSecrets(q)
}

func (d *datastoreImpl) SearchListSecrets(request *v1.Query) ([]*v1.ListSecret, error) {
	return d.searcher.SearchListSecrets(request)
}

func (d *datastoreImpl) CountSecrets() (int, error) {
	return d.storage.CountSecrets()
}

func (d *datastoreImpl) UpsertSecret(request *v1.Secret) error {
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
