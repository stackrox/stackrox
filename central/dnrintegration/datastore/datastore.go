package datastore

import (
	"bitbucket.org/stack-rox/apollo/central/dnrintegration"
	"bitbucket.org/stack-rox/apollo/central/dnrintegration/store"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/logging"
)

var (
	logger = logging.LoggerForModule()
)

// DataStore is an intermediary to the D&R integration storage.
// This is the entrypoint for all interaction with D&R.
//go:generate mockery -name=DataStore
type DataStore interface {
	GetDNRIntegration(id string) (*v1.DNRIntegration, bool, error)
	GetDNRIntegrations(request *v1.GetDNRIntegrationsRequest) ([]*v1.DNRIntegration, error)
	AddDNRIntegration(integration *v1.DNRIntegration) (string, error)
	UpdateDNRIntegration(integration *v1.DNRIntegration) error
	RemoveDNRIntegration(id string) error

	// ForCluster returns the DNRIntegration wrapper object for the given cluster ID if it exists.
	ForCluster(clusterID string) (integration dnrintegration.DNRIntegration, exists bool, err error)
}

// New returns an instance of DataStore
func New(store store.Store) DataStore {
	return &datastoreImpl{
		store: store,
	}
}
