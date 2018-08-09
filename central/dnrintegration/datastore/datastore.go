package datastore

import (
	"github.com/stackrox/rox/central/dnrintegration"
	"github.com/stackrox/rox/central/dnrintegration/store"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/logging"
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
