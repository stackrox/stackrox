package datastore

import (
	"fmt"

	"bitbucket.org/stack-rox/apollo/central/db"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/errorhelpers"
)

// ClusterDataStore is an intermediary to ClusterStorage.
type ClusterDataStore interface {
	db.ClusterStorage
}

// NewClusterDataStore provides a new instance of ClusterDataStore
func NewClusterDataStore(clusters db.ClusterStorage, deployments DeploymentDataStore, alerts AlertDataStore) ClusterDataStore {
	return &clusterDataStoreImpl{
		ClusterStorage: clusters,
		deployments:    deployments,
		alerts:         alerts,
	}
}

type clusterDataStoreImpl struct {
	// Default to storage implementations.
	db.ClusterStorage

	deployments DeploymentDataStore
	alerts      AlertDataStore
}

// RemoveCluster removes an cluster from the storage and the indexer
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
	existingAlerts, err := ds.alerts.GetAlerts(&v1.ListAlertsRequest{
		Stale:        []bool{false},
		DeploymentId: deployment.GetId(),
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
