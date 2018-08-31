package datastore

import (
	"github.com/stackrox/rox/central/secret/index"
	"github.com/stackrox/rox/central/secret/search"
	"github.com/stackrox/rox/central/secret/store"
	"github.com/stackrox/rox/generated/api/v1"
)

// DataStore is an intermediary to SecretStorage.
//go:generate mockery -name=DataStore
type DataStore interface {
	SearchSecrets(request *v1.RawQuery) ([]*v1.SearchResult, error)
	SearchRawSecrets(request *v1.RawQuery) ([]*v1.Secret, error)

	GetSecrets(request *v1.RawQuery) ([]*v1.Secret, error)
	UpsertSecret(request *v1.Secret) error
	RemoveSecret(id string) error
}

// New returns a new instance of DataStore using the input store, indexer, and searcher.
func New(storage store.Store, indexer index.Indexer, searcher search.Searcher) DataStore {
	return &datastoreImpl{
		storage:  storage,
		indexer:  indexer,
		searcher: searcher,
	}
}
