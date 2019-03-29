package datastore

import (
	"fmt"
	"time"

	"github.com/pkg/errors"
	alertDataStore "github.com/stackrox/rox/central/alert/datastore"
	"github.com/stackrox/rox/central/cluster/index"
	"github.com/stackrox/rox/central/cluster/store"
	deploymentDataStore "github.com/stackrox/rox/central/deployment/datastore"
	nodeStore "github.com/stackrox/rox/central/node/globalstore"
	notifierProcessor "github.com/stackrox/rox/central/notifier/processor"
	secretDataStore "github.com/stackrox/rox/central/secret/datastore"
	"github.com/stackrox/rox/central/sensor/service/connection"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/search"
)

const (
	connectionTerminationTimeout = 5 * time.Second
)

type datastoreImpl struct {
	indexer  index.Indexer
	storage  store.Store
	notifier notifierProcessor.Processor

	ads alertDataStore.DataStore
	dds deploymentDataStore.DataStore
	ns  nodeStore.GlobalStore
	ss  secretDataStore.DataStore
	cm  connection.Manager
}

func (ds *datastoreImpl) UpdateClusterStatus(id string, status *storage.ClusterStatus) error {
	return ds.storage.UpdateClusterStatus(id, status)
}

func (ds *datastoreImpl) buildIndex() error {
	clusters, err := ds.storage.GetClusters()
	if err != nil {
		return err
	}
	return ds.indexer.AddClusters(clusters)
}

// Search searches through the clusters
func (ds *datastoreImpl) Search(q *v1.Query) ([]search.Result, error) {
	return ds.indexer.Search(q)
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
	id, err := ds.storage.AddCluster(cluster)
	if err != nil {
		return "", err
	}
	return id, ds.indexer.AddCluster(cluster)
}

// GetCluster is a pass through function to the underlying storage.
func (ds *datastoreImpl) UpdateCluster(cluster *storage.Cluster) error {
	if err := ds.storage.UpdateCluster(cluster); err != nil {
		return err
	}
	return ds.indexer.AddCluster(cluster)
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
		return fmt.Errorf("unable to find cluster %q", id)
	}
	if err != nil {
		return err
	}

	if err := ds.storage.RemoveCluster(id); err != nil {
		return errors.Wrapf(err, "failed to remove cluster %q", id)
	}

	go ds.postRemoveCluster(cluster)
	return ds.indexer.DeleteCluster(id)
}

func (ds *datastoreImpl) postRemoveCluster(cluster *storage.Cluster) {
	// Terminate the cluster connection to prevent new data from being stored.
	if ds.cm != nil {
		if conn := ds.cm.GetConnection(cluster.GetId()); conn != nil {
			conn.Terminate(errors.New("cluster was deleted"))
			if !concurrency.WaitWithTimeout(conn.Stopped(), connectionTerminationTimeout) {
				errorhelpers.PanicOnDevelopmentf("connection to sensor from cluster %s not terminated after %v", cluster.GetId(), connectionTerminationTimeout)
			}
		}
	}

	// Fetch the deployments.
	deployments, err := ds.getDeployments(cluster)
	if err != nil {
		log.Errorf("failed to get deployments for removed cluster %s: %v", cluster.GetId(), err)
	}
	// Tombstone each deployment and mark alerts stale.
	for _, deployment := range deployments {
		alerts, err := ds.getAlerts(deployment)
		if err != nil {
			log.Errorf("failed to retrieve alerts for deployment %s: %v", deployment.GetId(), err)
		} else {
			err = ds.markAlertsStale(alerts)
			if err != nil {
				log.Errorf("failed to mark alerts for deployment %s as stale: %v", deployment.GetId(), err)
			}
		}

		err = ds.dds.RemoveDeployment(deployment.GetId())
		if err != nil {
			log.Errorf("failed to remove deployment %s in deleted cluster: %v", deployment.GetId(), err)
		}
	}

	// Remove nodes associated with this cluster
	if err := ds.ns.RemoveClusterNodeStore(cluster.GetId()); err != nil {
		log.Errorf("failed to remove nodes for cluster %s: %v", cluster.GetId(), err)
	}

	secrets, err := ds.getSecrets(cluster)
	if err != nil {
		log.Errorf("failed to obtain secrets in deleted cluster %s: %v", cluster.GetId(), err)
	}
	for _, s := range secrets {
		// Best effort to remove. If the object doesn't exist, then that is okay
		_ = ds.ss.RemoveSecret(s.GetId())
	}
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
func (ds *datastoreImpl) getAlerts(deployment *storage.ListDeployment) ([]*storage.Alert, error) {
	qb := search.NewQueryBuilder().AddStrings(search.ViolationState, storage.ViolationState_ACTIVE.String()).AddExactMatches(search.DeploymentID, deployment.GetId())

	existingAlerts, err := ds.ads.SearchRawAlerts(qb.ProtoQuery())
	if err != nil {
		log.Errorf("unable to get alert: %s", err)
		return nil, err
	}
	return existingAlerts, nil
}

func (ds *datastoreImpl) markAlertsStale(alerts []*storage.Alert) error {
	errorList := errorhelpers.NewErrorList("unable to mark some alerts stale")
	for _, alert := range alerts {
		errorList.AddError(ds.ads.MarkAlertStale(alert.GetId()))
		if errorList.ToError() == nil {
			// run notifier for all the resolved alerts
			ds.notifier.ProcessAlert(alert)
		}
	}
	return errorList.ToError()
}
