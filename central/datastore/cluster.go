package datastore

import (
	"fmt"

	"bitbucket.org/stack-rox/apollo/central/db"
	"bitbucket.org/stack-rox/apollo/central/search"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/errorhelpers"
)

// ClusterDataStore is an intermediary to ClusterStorage.
type ClusterDataStore interface {
	db.ClusterStorage
}

// NewClusterDataStore provides a new instance of ClusterDataStore
func NewClusterDataStore(clusters db.ClusterStorage, deployments DeploymentDataStore, alerts AlertDataStore,
	dnrIntegrations db.DNRIntegrationStorage) ClusterDataStore {
	return &clusterDataStoreImpl{
		ClusterStorage:  clusters,
		deployments:     deployments,
		alerts:          alerts,
		dnrIntegrations: dnrIntegrations,
	}
}

type clusterDataStoreImpl struct {
	// Default to storage implementations.
	db.ClusterStorage

	deployments     DeploymentDataStore
	alerts          AlertDataStore
	dnrIntegrations db.DNRIntegrationStorage
}

// If we remove a cluster, we remove the DNR integration from it, if there is one.
func (ds *clusterDataStoreImpl) removeDNRIntegrationIfExists(clusterID string) error {
	dnrIntegrations, err := ds.dnrIntegrations.GetDNRIntegrations(&v1.GetDNRIntegrationsRequest{
		ClusterId: clusterID,
	})
	if err != nil {
		return fmt.Errorf("fetching DNR integrations: %s", err)
	}

	// There should be either 0 or 1 DNR integrations, but this is not the place to assert that.
	// We'll just log a message here, and remove all the integrations.
	if len(dnrIntegrations) > 1 {
		logger.Errorf("Found more than 1 D&R integration for cluster %s: %#v",
			clusterID, dnrIntegrations)
	}

	for _, integration := range dnrIntegrations {
		err = ds.dnrIntegrations.RemoveDNRIntegration(integration.GetId())
		if err != nil {
			return fmt.Errorf("removing DNR integration %s: %s", integration.GetId(), err)
		}
	}
	return nil
}

// RemoveCluster removes a cluster from the storage and the indexer
func (ds *clusterDataStoreImpl) RemoveCluster(id string) error {
	// Fetch the cluster an confirm it exists.
	cluster, exists, err := ds.ClusterStorage.GetCluster(id)
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

		err = ds.deployments.RemoveDeployment(deployment.GetId())
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

	return ds.ClusterStorage.RemoveCluster(id)
}

// TODO(cgorman) Make this a search once the document mapping goes in
// RemoveCluster removes an cluster from the storage and the indexer
func (ds *clusterDataStoreImpl) getDeployments(cluster *v1.Cluster) ([]*v1.Deployment, error) {
	deployments, err := ds.deployments.GetDeployments()
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
func (ds *clusterDataStoreImpl) getAlerts(deployment *v1.Deployment) ([]*v1.Alert, error) {
	qb := search.NewQueryBuilder().AddBool(search.Stale, false).AddString(search.DeploymentID, deployment.GetId())

	existingAlerts, err := ds.alerts.GetAlerts(&v1.ListAlertsRequest{
		Query: qb.Query(),
	})
	if err != nil {
		logger.Errorf("unable to get alert: %s", err)
		return nil, err
	}
	return existingAlerts, nil
}

func (ds *clusterDataStoreImpl) markAlertsStale(alerts []*v1.Alert) error {
	var errors []error
	for _, alert := range alerts {
		alert.Stale = true
		if err := ds.alerts.UpdateAlert(alert); err != nil {
			errors = append(errors, err)
		}
	}
	return errorhelpers.FormatErrors("unable to mark some alerts stale", errors)
}
