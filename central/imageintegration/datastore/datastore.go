package datastore

import (
	"bitbucket.org/stack-rox/apollo/central/imageintegration/store"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

// DataStore is the entry point for modifying Cluster data.
type DataStore interface {
	GetImageIntegration(id string) (*v1.ImageIntegration, bool, error)
	GetImageIntegrations(integration *v1.GetImageIntegrationsRequest) ([]*v1.ImageIntegration, error)
	AddImageIntegration(integration *v1.ImageIntegration) (string, error)
	UpdateImageIntegration(integration *v1.ImageIntegration) error
	RemoveImageIntegration(id string) error
}

// New returns an instance of DataStore.
func New(storage store.Store) DataStore {
	return &datastoreImpl{
		storage: storage,
	}
}
