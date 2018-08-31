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

func (d *datastoreImpl) SearchSecrets(request *v1.RawQuery) ([]*v1.SearchResult, error) {
	return d.searcher.SearchSecrets(request)
}

func (d *datastoreImpl) SearchRawSecrets(request *v1.RawQuery) ([]*v1.Secret, error) {
	return d.searcher.SearchRawSecrets(request)
}

func (d *datastoreImpl) GetSecrets(request *v1.RawQuery) ([]*v1.Secret, error) {
	return d.searcher.SearchRawSecrets(request)
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
