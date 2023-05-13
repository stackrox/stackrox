package datastore

import (
	"context"

	"github.com/stackrox/rox/central/continuousintegration/store"
	"github.com/stackrox/rox/generated/storage"
)

// DataStore for continuous integration configs.
type DataStore interface {
	GetContinuousIntegrationConfig(ctx context.Context, id string) (*storage.ContinuousIntegrationConfig, bool, error)
	GetAllContinuousIntegrationConfigs(ctx context.Context) ([]*storage.ContinuousIntegrationConfig, error)
	AddContinuousIntegrationConfig(ctx context.Context, config *storage.ContinuousIntegrationConfig) (*storage.ContinuousIntegrationConfig, error)
	UpdateContinuousIntegrationConfig(ctx context.Context, config *storage.ContinuousIntegrationConfig) error
	RemoveContinuousIntegrationConfig(ctx context.Context, id string) error
}

// New returns an instance of DataStore.
func New(store store.ContinuousIntegrationStore) DataStore {
	return &dataStoreImpl{store: store}
}
