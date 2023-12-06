package datastore

import (
	"context"

	"github.com/stackrox/rox/central/integrationhealth/store"
	"github.com/stackrox/rox/generated/storage"
)

// DataStore is the entry point for modifying integration health data.
//
//go:generate mockgen-wrapper
type DataStore interface {
	GetRegistriesAndScanners(ctx context.Context) ([]*storage.IntegrationHealth, error)
	GetNotifierPlugins(ctx context.Context) ([]*storage.IntegrationHealth, error)
	GetBackupPlugins(ctx context.Context) ([]*storage.IntegrationHealth, error)
	GetDeclarativeConfigs(ctx context.Context) ([]*storage.IntegrationHealth, error)

	UpsertIntegrationHealth(ctx context.Context, integrationHealth *storage.IntegrationHealth) error
	RemoveIntegrationHealth(ctx context.Context, id string) error
	GetIntegrationHealth(ctx context.Context, id string) (*storage.IntegrationHealth, bool, error)
}

// New returns an instance of DataStore.
func New(storage store.Store) DataStore {
	return &datastoreImpl{
		store: storage,
	}
}
