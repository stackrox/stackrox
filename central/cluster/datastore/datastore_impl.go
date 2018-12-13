package datastore

import (
	"fmt"
	"time"

	alertDataStore "github.com/stackrox/rox/central/alert/datastore"
	"github.com/stackrox/rox/central/cluster/store"
	deploymentDataStore "github.com/stackrox/rox/central/deployment/datastore"
	nodeStore "github.com/stackrox/rox/central/node/store"
	secretDataStore "github.com/stackrox/rox/central/secret/datastore"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/search"
)

type datastoreImpl struct {
	storage store.Store

	ads alertDataStore.DataStore
	dds deploymentDataStore.DataStore
	ns  nodeStore.GlobalStore
	ss  secretDataStore.DataStore
}

// GetCluster is a pass through function to the underlying storage.
func (ds *datastoreImpl) GetCluster(id string) (*storage.Cluster, bool, error) {
	return ds.storage.GetCluster(id)
}

// GetCluster is a pass through function to the underlying storage.
func (ds *datastoreImpl) GetClusters() ([]*storage.Cluster, error) {
	return ds.storage.GetClusters()
}

// GetCluster is a pass through function to the underlying storage.
func (ds *datastoreImpl) CountClusters() (int, error) {
	return ds.storage.CountClusters()
}

// GetCluster is a pass through function to the underlying storage.
func (ds *datastoreImpl) AddCluster(cluster *storage.Cluster) (string, error) {
	return ds.storage.AddCluster(cluster)
}

// GetCluster is a pass through function to the underlying storage.
func (ds *datastoreImpl) UpdateCluster(cluster *storage.Cluster) error {
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

	// Remove nodes associated with this cluster
	if err := ds.ns.RemoveClusterNodeStore(id); err != nil {
		return err
	}

	secrets, err := ds.getSecrets(cluster)
	if err != nil {
		return err
	}
	for _, s := range secrets {
		// Best effort to remove. If the object doesn't exist, then that is okay
		ds.ss.RemoveSecret(s.GetId())
	}

	return ds.storage.RemoveCluster(id)
}

// RemoveCluster removes an cluster from the storage and the indexer
func (ds *datastoreImpl) getSecrets(cluster *storage.Cluster) ([]*storage.ListSecret, error) {
	q := search.NewQueryBuilder().AddExactMatches(search.ClusterID, cluster.GetId()).ProtoQuery()
	return ds.ss.SearchListSecrets(q)
}

// RemoveCluster removes an cluster from the storage and the indexer
func (ds *datastoreImpl) getDeployments(cluster *storage.Cluster) ([]*storage.ListDeployment, error) {
	deployments, err := ds.dds.ListDeployments()
	if err != nil {
		return nil, err
	}

	wantedDeployments := make([]*storage.ListDeployment, 0)
	for _, d := range deployments {
		if d.GetClusterId() == cluster.GetId() {
			wantedDeployments = append(wantedDeployments, d)
		}
	}
	return wantedDeployments, nil
}

// TODO(cgorman) Make this a search once the document mapping goes in
func (ds *datastoreImpl) getAlerts(deployment *storage.ListDeployment) ([]*v1.ListAlert, error) {
	qb := search.NewQueryBuilder().AddStrings(search.ViolationState, v1.ViolationState_ACTIVE.String()).AddExactMatches(search.DeploymentID, deployment.GetId())

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

// UpdateMetadata updates the cluster with cloud provider metadata
func (ds *datastoreImpl) UpdateMetadata(id string, metadata *storage.ProviderMetadata) error {
	return ds.storage.UpdateMetadata(id, metadata)
}
