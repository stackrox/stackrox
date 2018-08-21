package datastore

import (
	"github.com/gogo/protobuf/proto"
	deploymentDataStore "github.com/stackrox/rox/central/deployment/datastore"
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
	AddDNRIntegration(proto *v1.DNRIntegration, integration dnrintegration.DNRIntegration) (string, error)
	UpdateDNRIntegration(proto *v1.DNRIntegration, integration dnrintegration.DNRIntegration) error
	RemoveDNRIntegration(id string) error

	// ForCluster returns the DNRIntegration wrapper object for the given cluster ID if it exists.
	ForCluster(clusterID string) (integration dnrintegration.DNRIntegration, exists bool)
}

// New returns an instance of DataStore.
func New(store store.Store, deploymentStore deploymentDataStore.DataStore) (DataStore, error) {
	protoIntegrations, err := store.GetDNRIntegrations(&v1.GetDNRIntegrationsRequest{})
	if err != nil {
		return nil, err
	}

	d := &datastoreImpl{
		store: store,
		clusterToIntegrations: make(map[string]dnrintegration.DNRIntegration),
	}

	for _, protoIntegration := range protoIntegrations {
		integration, err := dnrintegration.New(protoIntegration, deploymentStore)
		// If, on restart, we can't connect to D&R, don't panic; just remove the integration.
		if err != nil {
			logger.Errorf("Failed to create D&R integration %s: %s", proto.MarshalTextString(protoIntegration), err)
			err := store.RemoveDNRIntegration(protoIntegration.GetId())
			if err != nil {
				logger.Errorf("Failed to remove D&R integration %s from store: %s", protoIntegration.GetId(), err)
			}
			continue
		}
		d.updateMap(protoIntegration.GetClusterIds(), integration)
	}
	return d, nil
}
