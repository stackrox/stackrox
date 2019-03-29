package datastore

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/secret/index"
	"github.com/stackrox/rox/central/secret/search"
	"github.com/stackrox/rox/central/secret/store"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	searchPkg "github.com/stackrox/rox/pkg/search"
)

// DataStore is an intermediary to SecretStorage.
//go:generate mockgen-wrapper DataStore
type DataStore interface {
	Search(q *v1.Query) ([]searchPkg.Result, error)
	SearchSecrets(q *v1.Query) ([]*v1.SearchResult, error)
	SearchListSecrets(q *v1.Query) ([]*storage.ListSecret, error)

	CountSecrets() (int, error)
	ListSecrets() ([]*storage.ListSecret, error)
	GetSecret(id string) (*storage.Secret, bool, error)
	UpsertSecret(request *storage.Secret) error
	RemoveSecret(id string) error
}

// New returns a new instance of DataStore using the input store, indexer, and searcher.
func New(storage store.Store, indexer index.Indexer, searcher search.Searcher) (DataStore, error) {
	d := &datastoreImpl{
		storage:  storage,
		indexer:  indexer,
		searcher: searcher,
	}
	if err := d.buildIndex(); err != nil {
		return nil, errors.Wrap(err, "failed to build index from existing store")
	}
	return d, nil
}
