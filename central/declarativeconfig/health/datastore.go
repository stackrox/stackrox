package health

import (
	"context"

	"github.com/stackrox/rox/central/declarativeconfig/health/store"
	"github.com/stackrox/rox/generated/storage"
)

// DataStore is the entry point for modifying declarative config health data.
//
//go:generate mockgen-wrapper
type DataStore interface {
	GetDeclarativeConfigs(ctx context.Context) ([]*storage.DeclarativeConfigHealth, error)
	UpsertDeclarativeConfig(ctx context.Context, configHealth *storage.DeclarativeConfigHealth) error
	RemoveDeclarativeConfig(ctx context.Context, id string) error
	GetDeclarativeConfig(ctx context.Context, id string) (*storage.DeclarativeConfigHealth, bool, error)
}

func New(storage store.Store) DataStore {
	return &datastoreImpl{
		store: storage,
	}
}
