package datastore

import (
	"fmt"

	"github.com/stackrox/rox/central/dnrintegration"
	"github.com/stackrox/rox/central/dnrintegration/store"
	"github.com/stackrox/rox/generated/api/v1"
)

type datastoreImpl struct {
	store store.Store
}

func (d *datastoreImpl) ForCluster(clusterID string) (integration dnrintegration.DNRIntegration, exists bool, err error) {
	integrations, err := d.store.GetDNRIntegrations(&v1.GetDNRIntegrationsRequest{ClusterId: clusterID})
	if err != nil {
		err = fmt.Errorf("failed to retrieve integrations for cluster %s: %s", clusterID, err)
		return
	}

	if len(integrations) == 0 {
		return
	}

	exists = true

	// This should never happen, but it's counter-productive to return an error here.
	// Simply log the error for now.
	if len(integrations) > 1 {
		logger.Errorf("Found multiple integrations for cluster %s", clusterID)
	}

	integration, err = dnrintegration.New(integrations[0])
	if err != nil {
		err = fmt.Errorf("bad DNR integration for cluster %s: %s", clusterID, err)
		return
	}
	return
}

// GetDNRIntegration is a pass-through to the underlying store's function.
func (d *datastoreImpl) GetDNRIntegration(id string) (*v1.DNRIntegration, bool, error) {
	return d.store.GetDNRIntegration(id)
}

// GetDNRIntegrations is a pass-through to the underlying store's function.
func (d *datastoreImpl) GetDNRIntegrations(request *v1.GetDNRIntegrationsRequest) ([]*v1.DNRIntegration, error) {
	return d.store.GetDNRIntegrations(request)
}

// AddDNRIntegration is a pass-through to the underlying store's function.
func (d *datastoreImpl) AddDNRIntegration(integration *v1.DNRIntegration) (string, error) {
	return d.store.AddDNRIntegration(integration)
}

// UpdateDNRIntegration is a pass-through to the underlying store's function.
func (d *datastoreImpl) UpdateDNRIntegration(integration *v1.DNRIntegration) error {
	return d.store.UpdateDNRIntegration(integration)
}

// RemoveDNRIntegration is a pass-through to the underlying store's function.
func (d *datastoreImpl) RemoveDNRIntegration(id string) error {
	return d.store.RemoveDNRIntegration(id)
}
