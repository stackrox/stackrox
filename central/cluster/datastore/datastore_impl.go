package datastore

import (
	"fmt"
	"time"

	alertDataStore "bitbucket.org/stack-rox/apollo/central/alert/datastore"
	"bitbucket.org/stack-rox/apollo/central/cluster/store"
	deploymentDataStore "bitbucket.org/stack-rox/apollo/central/deployment/datastore"
	dnrStore "bitbucket.org/stack-rox/apollo/central/dnrintegration/store"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/errorhelpers"
	"bitbucket.org/stack-rox/apollo/pkg/search"
)

type datastoreImpl struct {
	storage store.Store

	ads alertDataStore.DataStore
	dds deploymentDataStore.DataStore
	dnr dnrStore.Store
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
	var errors []error
	for _, deployment := range deployments {
		alerts, err := ds.getAlerts(deployment)
		if err != nil {
			errors = append(errors, err)
			continue
		}

		err = ds.markAlertsStale(alerts)
		if err != nil {
			errors = append(errors, err)
			continue
		}

		err = ds.dds.RemoveDeployment(deployment.GetId())
		if err != nil {
			errors = append(errors, err)
			continue
		}
	}
	if len(errors) > 0 {
		return errorhelpers.FormatErrors("unable to complete cluster removal", errors)
	}

	err = ds.removeDNRIntegrationIfExists(id)
	if err != nil {
		return err
	}

	return ds.storage.RemoveCluster(id)
}

// TODO(cgorman) Make this a search once the document mapping goes in
// RemoveCluster removes an cluster from the storage and the indexer
func (ds *datastoreImpl) getDeployments(cluster *v1.Cluster) ([]*v1.Deployment, error) {
	deployments, err := ds.dds.GetDeployments()
	if err != nil {
		return nil, err
	}

	wantedDeployments := make([]*v1.Deployment, 0)
	for _, d := range deployments {
		if d.GetClusterId() == cluster.Id {
			wantedDeployments = append(wantedDeployments, d)
		}
	}
	return wantedDeployments, nil
}

// TODO(cgorman) Make this a search once the document mapping goes in
func (ds *datastoreImpl) getAlerts(deployment *v1.Deployment) ([]*v1.Alert, error) {
	qb := search.NewQueryBuilder().AddBool(search.Stale, false).AddString(search.DeploymentID, deployment.GetId())

	existingAlerts, err := ds.ads.GetAlerts(&v1.ListAlertsRequest{
		Query: qb.Query(),
	})
	if err != nil {
		log.Errorf("unable to get alert: %s", err)
		return nil, err
	}
	return existingAlerts, nil
}

func (ds *datastoreImpl) markAlertsStale(alerts []*v1.Alert) error {
	var errors []error
	for _, alert := range alerts {
		alert.Stale = true
		if err := ds.ads.UpdateAlert(alert); err != nil {
			errors = append(errors, err)
		}
	}
	return errorhelpers.FormatErrors("unable to mark some alerts stale", errors)
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
