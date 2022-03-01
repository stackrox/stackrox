package datastore

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/secret/internal/index"
	"github.com/stackrox/rox/central/secret/internal/store"
	"github.com/stackrox/rox/central/secret/search"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	searchPkg "github.com/stackrox/rox/pkg/search"
)

// DataStore is an intermediary to SecretStorage.
//go:generate mockgen-wrapper
type DataStore interface {
	Search(ctx context.Context, q *v1.Query) ([]searchPkg.Result, error)
	Count(ctx context.Context, q *v1.Query) (int, error)
	SearchSecrets(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error)
	SearchRawSecrets(ctx context.Context, q *v1.Query) ([]*storage.Secret, error)
	SearchListSecrets(ctx context.Context, q *v1.Query) ([]*storage.ListSecret, error)

	CountSecrets(ctx context.Context) (int, error)
	GetSecret(ctx context.Context, id string) (*storage.Secret, bool, error)
	UpsertSecret(ctx context.Context, request *storage.Secret) error
	RemoveSecret(ctx context.Context, id string) error
}

// New returns a new instance of DataStore using the input store, indexer, and searcher.
func New(storage store.Store, indexer index.Indexer, searcher search.Searcher) (DataStore, error) {
	d := &datastoreImpl{
		storage:  storage,
		indexer:  indexer,
		searcher: searcher,
	}
	if err := d.buildIndex(context.TODO()); err != nil {
		return nil, errors.Wrap(err, "failed to build index from existing store")
	}
	return d, nil
}
