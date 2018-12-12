package datastore

import (
	"github.com/stackrox/rox/central/imageintegration/store"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

// DataStore is the entry point for modifying Cluster data.
type DataStore interface {
	GetImageIntegration(id string) (*storage.ImageIntegration, bool, error)
	GetImageIntegrations(integration *v1.GetImageIntegrationsRequest) ([]*storage.ImageIntegration, error)

	AddImageIntegration(integration *storage.ImageIntegration) (string, error)
	UpdateImageIntegration(integration *storage.ImageIntegration) error
	RemoveImageIntegration(id string) error
}

// New returns an instance of DataStore.
func New(storage store.Store) DataStore {
	return &datastoreImpl{
		storage: storage,
	}
}
