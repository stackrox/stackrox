package datastore

import (
	"context"

	"github.com/stackrox/rox/central/logintegrations/datastore/internal/store"
	"github.com/stackrox/rox/generated/storage"
)

// DataStore is an intermediary to LogIntegrations storage.
//go:generate mockgen-wrapper
type DataStore interface {
	GetLogIntegration(ctx context.Context, id string) (*storage.LogIntegration, bool, error)
	GetLogIntegrations(ctx context.Context) ([]*storage.LogIntegration, error)
	CreateLogIntegration(ctx context.Context, integration *storage.LogIntegration) error
	UpdateLogIntegration(ctx context.Context, integration *storage.LogIntegration) error
	DeleteLogIntegration(ctx context.Context, id string) error
}

// New returns an instance of DataStore.
func New(storage store.Store) DataStore {
	return &datastoreImpl{
		storage: storage,
	}
}
