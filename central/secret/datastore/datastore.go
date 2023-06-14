package datastore

import (
	"context"
	"testing"

	"github.com/stackrox/rox/central/secret/internal/store"
	pgStore "github.com/stackrox/rox/central/secret/internal/store/postgres"
	"github.com/stackrox/rox/central/secret/search"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres"
	searchPkg "github.com/stackrox/rox/pkg/search"
)

// DataStore is an intermediary to SecretStorage.
//
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

// New returns a new instance of DataStore using the input store, and searcher.
func New(secretStore store.Store, searcher search.Searcher) (DataStore, error) {
	d := &datastoreImpl{
		storage:  secretStore,
		searcher: searcher,
	}
	return d, nil
}

// GetTestPostgresDataStore provides a datastore connected to postgres for testing purposes.
func GetTestPostgresDataStore(_ *testing.T, pool postgres.DB) (DataStore, error) {
	dbstore := pgStore.New(pool)
	indexer := pgStore.NewIndexer(pool)
	searcher := search.New(dbstore, indexer)
	return New(dbstore, searcher)
}
