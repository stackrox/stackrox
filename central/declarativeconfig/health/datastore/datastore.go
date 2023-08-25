package datastore

import (
	"context"

	"github.com/stackrox/rox/central/declarativeconfig/health/datastore/store"
	"github.com/stackrox/rox/generated/storage"
)

// DataStore is the entry point for modifying declarative config health data.
//
//go:generate mockgen-wrapper
type DataStore interface {
	GetDeclarativeConfigs(ctx context.Context) ([]*storage.DeclarativeConfigHealth, error)
	UpsertDeclarativeConfig(ctx context.Context, configHealth *storage.DeclarativeConfigHealth) error
	UpdateStatusForDeclarativeConfig(ctx context.Context, id string, err error) error
	RemoveDeclarativeConfig(ctx context.Context, id string) error
	GetDeclarativeConfig(ctx context.Context, id string) (*storage.DeclarativeConfigHealth, bool, error)
}

// New returns new instance of declarative config health datastore.
func New(storage store.Store) DataStore {
	return &datastoreImpl{
		store: storage,
	}
}
