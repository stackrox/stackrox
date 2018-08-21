package datastore

import (
	"fmt"
	"sync"

	"github.com/stackrox/rox/central/dnrintegration"
	"github.com/stackrox/rox/central/dnrintegration/store"
	"github.com/stackrox/rox/generated/api/v1"
)

type datastoreImpl struct {
	store store.Store

	clusterToIntegrations map[string]dnrintegration.DNRIntegration
	lock                  sync.RWMutex
}

func (d *datastoreImpl) ForCluster(clusterID string) (integration dnrintegration.DNRIntegration, exists bool) {
	d.lock.RLock()
	defer d.lock.RUnlock()
	integration, exists = d.clusterToIntegrations[clusterID]
	return
}

func (d *datastoreImpl) GetDNRIntegration(id string) (*v1.DNRIntegration, bool, error) {
	return d.store.GetDNRIntegration(id)
}

func (d *datastoreImpl) GetDNRIntegrations(request *v1.GetDNRIntegrationsRequest) ([]*v1.DNRIntegration, error) {
	return d.store.GetDNRIntegrations(request)
}

func (d *datastoreImpl) updateMap(clusterIDs []string, integration dnrintegration.DNRIntegration) {
	d.lock.Lock()
	defer d.lock.Unlock()
	for _, clusterID := range clusterIDs {
		d.clusterToIntegrations[clusterID] = integration
	}
}

func (d *datastoreImpl) AddDNRIntegration(proto *v1.DNRIntegration, integration dnrintegration.DNRIntegration) (string, error) {
	id, err := d.store.AddDNRIntegration(proto)
	if err != nil {
		return "", err
	}

	d.updateMap(proto.GetClusterIds(), integration)
	return id, nil
}

func (d *datastoreImpl) UpdateDNRIntegration(proto *v1.DNRIntegration, integration dnrintegration.DNRIntegration) error {
	err := d.store.UpdateDNRIntegration(proto)
	if err != nil {
		return err
	}
	d.updateMap(proto.GetClusterIds(), integration)
	return nil
}

func (d *datastoreImpl) RemoveDNRIntegration(id string) error {
	integration, exists, err := d.store.GetDNRIntegration(id)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("D&R integration '%s' not found", id)
	}
	err = d.store.RemoveDNRIntegration(id)
	if err != nil {
		return err
	}
	d.lock.Lock()
	defer d.lock.Unlock()
	for _, clusterID := range integration.GetClusterIds() {
		delete(d.clusterToIntegrations, clusterID)
	}
	return nil
}
