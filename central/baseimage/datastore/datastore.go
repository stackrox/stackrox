package datastore

import (
	"context"

	biStr "github.com/stackrox/rox/central/baseimage/store/postgres"
	bilStr "github.com/stackrox/rox/central/baseimagelayer/store/postgres"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres"
)

type DataStore interface {
	GetCandidateLayers(ctx context.Context, layerSHA string) ([]*storage.BaseImageLayer, bool, error)
}

// New returns a new instance of a DataStore.
func New(db postgres.DB, baseImageStore biStr.Store, baseImageLayerStore bilStr.Store) DataStore {
	return &datastoreImpl{
		db:                  db,
		baseImageStore:      baseImageStore,
		baseImageLayerStore: baseImageLayerStore,
	}
}
