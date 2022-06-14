package datastore

import (
	"context"

	"github.com/stackrox/stackrox/central/imageintegration/store"
	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

// DataStore is the entry point for modifying Cluster data.
//go:generate mockgen-wrapper
type DataStore interface {
	GetImageIntegration(ctx context.Context, id string) (*storage.ImageIntegration, bool, error)
	GetImageIntegrations(ctx context.Context, integration *v1.GetImageIntegrationsRequest) ([]*storage.ImageIntegration, error)

	AddImageIntegration(ctx context.Context, integration *storage.ImageIntegration) (string, error)
	UpdateImageIntegration(ctx context.Context, integration *storage.ImageIntegration) error
	RemoveImageIntegration(ctx context.Context, id string) error
}

// New returns an instance of DataStore.
func New(storage store.Store) DataStore {
	return &datastoreImpl{
		storage: storage,
	}
}
