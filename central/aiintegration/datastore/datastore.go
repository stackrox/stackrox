package datastore

import (
	"context"
	"testing"

	"github.com/stackrox/rox/central/aiintegration/datastore/internal/store"
	pgStore "github.com/stackrox/rox/central/aiintegration/datastore/internal/store/postgres"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres"
)

// DataStore is the datastore for AI integrations.
//
//go:generate mockgen-wrapper
type DataStore interface {
	Add(ctx context.Context, integration *storage.AiIntegration) (string, error)
	Get(ctx context.Context, id string) (*storage.AiIntegration, bool, error)
	GetAll(ctx context.Context) ([]*storage.AiIntegration, error)
	Upsert(ctx context.Context, integration *storage.AiIntegration) error
	Delete(ctx context.Context, id string) error
	Exists(ctx context.Context, id string) (bool, error)
}

// New returns a new DataStore instance.
func New(s store.Store) DataStore {
	return &dataStore{
		store: s,
	}
}

// GetTestPostgresDataStore provides a datastore connected to postgres for testing purposes.
func GetTestPostgresDataStore(_ testing.TB, pool postgres.DB) DataStore {
	return New(pgStore.New(pool))
}
