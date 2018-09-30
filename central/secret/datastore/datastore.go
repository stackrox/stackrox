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
	SearchSecrets(q *v1.Query) ([]*v1.SearchResult, error)
	SearchListSecrets(q *v1.Query) ([]*v1.ListSecret, error)

	CountSecrets() (int, error)
	GetSecret(id string) (*v1.Secret, bool, error)
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
