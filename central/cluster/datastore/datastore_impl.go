package datastore

import (
	"fmt"
	"time"

	alertDataStore "github.com/stackrox/rox/central/alert/datastore"
	"github.com/stackrox/rox/central/cluster/store"
	deploymentDataStore "github.com/stackrox/rox/central/deployment/datastore"
	dnrStore "github.com/stackrox/rox/central/dnrintegration/datastore"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/search"
)

type datastoreImpl struct {
	storage store.Store

	ads alertDataStore.DataStore
	dds deploymentDataStore.DataStore
	dnr dnrStore.DataStore
}

// GetCluster is a pass through function to the underlying storage.
func (ds *datastoreImpl) GetCluster(id string) (*v1.Cluster, bool, error) {
	return ds.storage.GetCluster(id)
}

// GetCluster is a pass through function to the underlying storage.
func (ds *datastoreImpl) GetClusters() ([]*v1.Cluster, error) {
	return ds.storage.GetClusters()
}

// GetCluster is a pass through function to the underlying storage.
func (ds *datastoreImpl) CountClusters() (int, error) {
	return ds.storage.CountClusters()
}

// GetCluster is a pass through function to the underlying storage.
func (ds *datastoreImpl) AddCluster(cluster *v1.Cluster) (string, error) {
	return ds.storage.AddCluster(cluster)
}

// GetCluster is a pass through function to the underlying storage.
func (ds *datastoreImpl) UpdateCluster(cluster *v1.Cluster) error {
	return ds.storage.UpdateCluster(cluster)
}

// GetCluster is a pass through function to the underlying storage.
func (ds *datastoreImpl) UpdateClusterContactTime(id string, t time.Time) error {
	return ds.storage.UpdateClusterContactTime(id, t)
}

// RemoveCluster removes a cluster from the storage and the indexer
func (ds *datastoreImpl) RemoveCluster(id string) error {
	// Fetch the cluster an confirm it exists.
	cluster, exists, err := ds.storage.GetCluster(id)
	if !exists {
		return fmt.Errorf("unable to find cluster %s", id)
	}
	if err != nil {
		return err
	}

	// Fetch the deployments.
	deployments, err := ds.getDeployments(cluster)
	if err != nil {
		return err
	}

	// Tombstone each deployment and mark alerts stale.
	errorList := errorhelpers.NewErrorList("unable to complete cluster removal")
	for _, deployment := range deployments {
		alerts, err := ds.getAlerts(deployment)
		if err != nil {
			errorList.AddError(err)
			continue
		}

		err = ds.markAlertsStale(alerts)
		if err != nil {
			errorList.AddError(err)
			continue
		}

		err = ds.dds.RemoveDeployment(deployment.GetId())
		if err != nil {
			errorList.AddError(err)
			continue
		}
	}
	if err := errorList.ToError(); err != nil {
		return err
	}

	err = ds.removeDNRIntegrationIfExists(id)
	if err != nil {
		return err
	}

	return ds.storage.RemoveCluster(id)
}

// RemoveCluster removes an cluster from the storage and the indexer
func (ds *datastoreImpl) getDeployments(cluster *v1.Cluster) ([]*v1.ListDeployment, error) {
	deployments, err := ds.dds.ListDeployments()
	if err != nil {
		return nil, err
	}

	wantedDeployments := make([]*v1.ListDeployment, 0)
	for _, d := range deployments {
		if d.GetClusterId() == cluster.GetId() {
			wantedDeployments = append(wantedDeployments, d)
		}
	}
	return wantedDeployments, nil
}

// TODO(cgorman) Make this a search once the document mapping goes in
func (ds *datastoreImpl) getAlerts(deployment *v1.ListDeployment) ([]*v1.ListAlert, error) {
	qb := search.NewQueryBuilder().AddBools(search.Stale, false).AddStrings(search.DeploymentID, deployment.GetId())

	existingAlerts, err := ds.ads.ListAlerts(&v1.ListAlertsRequest{
		Query: qb.Query(),
	})
	if err != nil {
		log.Errorf("unable to get alert: %s", err)
		return nil, err
	}
	return existingAlerts, nil
}

func (ds *datastoreImpl) markAlertsStale(alerts []*v1.ListAlert) error {
	errorList := errorhelpers.NewErrorList("unable to mark some alerts stale")
	for _, alert := range alerts {
		errorList.AddError(ds.ads.MarkAlertStale(alert.GetId()))
	}
	return errorList.ToError()
}

// If we remove a cluster, we remove the DNR integration from it, if there is one.
func (ds *datastoreImpl) removeDNRIntegrationIfExists(clusterID string) error {
	dnrIntegrations, err := ds.dnr.GetDNRIntegrations(&v1.GetDNRIntegrationsRequest{
		ClusterId: clusterID,
	})
	if err != nil {
		return fmt.Errorf("fetching DNR integrations: %s", err)
	}

	// There should be either 0 or 1 DNR integrations, but this is not the place to assert that.
	// We'll just log a message here, and remove all the integrations.
	if len(dnrIntegrations) > 1 {
		log.Errorf("Found more than 1 D&R integration for cluster %s: %#v",
			clusterID, dnrIntegrations)
	}

	for _, integration := range dnrIntegrations {
		err = ds.dnr.RemoveDNRIntegration(integration.GetId())
		if err != nil {
			return fmt.Errorf("removing DNR integration %s: %s", integration.GetId(), err)
		}
	}
	return nil
}
