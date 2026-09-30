package datastore

import (
	"context"
	"testing"

	"github.com/stackrox/rox/central/aimetadata/datastore/internal/store"
	pgStore "github.com/stackrox/rox/central/aimetadata/datastore/internal/store/postgres"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/search"
)

// DataStore provides access to AI metadata associated with custom resources.
// Callers must authorise access through the corresponding CustomResource.
//
//go:generate mockgen-wrapper
type DataStore interface {
	GetAIMetadata(ctx context.Context, id string) (*storage.AIMetadata, bool, error)
	ListAIMetadata(ctx context.Context, query *v1.Query) ([]*storage.AIMetadata, error)
	Search(ctx context.Context, query *v1.Query) ([]search.Result, error)
	UpsertAIMetadata(ctx context.Context, metadata *storage.AIMetadata) error
	DeleteAIMetadata(ctx context.Context, id string) error
}

func newDataStore(store store.Store) DataStore {
	return &datastoreImpl{store: store}
}

// GetTestPostgresDataStore provides a datastore connected to Postgres for testing.
func GetTestPostgresDataStore(_ testing.TB, db postgres.DB) DataStore {
	return newDataStore(pgStore.New(db))
}
