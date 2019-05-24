package datastore

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	alertDataStore "github.com/stackrox/rox/central/alert/datastore"
	"github.com/stackrox/rox/central/cluster/index"
	"github.com/stackrox/rox/central/cluster/index/mappings"
	"github.com/stackrox/rox/central/cluster/store"
	deploymentDataStore "github.com/stackrox/rox/central/deployment/datastore"
	nodeDataStore "github.com/stackrox/rox/central/node/globaldatastore"
	notifierProcessor "github.com/stackrox/rox/central/notifier/processor"
	"github.com/stackrox/rox/central/role/resources"
	secretDataStore "github.com/stackrox/rox/central/secret/datastore"
	"github.com/stackrox/rox/central/sensor/service/connection"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/set"
)

const (
	connectionTerminationTimeout = 5 * time.Second
)

var (
	clusterSAC             = sac.ForResource(resources.Cluster)
	clusterSACSearchHelper = clusterSAC.MustCreateSearchHelper(mappings.OptionsMap, sac.ClusterIDField)
)

type datastoreImpl struct {
	indexer  index.Indexer
	storage  store.Store
	notifier notifierProcessor.Processor

	ads alertDataStore.DataStore
	dds deploymentDataStore.DataStore
	ns  nodeDataStore.GlobalDataStore
	ss  secretDataStore.DataStore
	cm  connection.Manager
}

func (ds *datastoreImpl) UpdateClusterStatus(ctx context.Context, id string, status *storage.ClusterStatus) error {
	if ok, err := clusterSAC.WriteAllowed(ctx, sac.ClusterScopeKey(id)); err != nil {
		return err
	} else if !ok {
		return errors.New("permission denied")
	}

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
func (ds *datastoreImpl) Search(ctx context.Context, q *v1.Query) ([]search.Result, error) {
	return clusterSACSearchHelper.Apply(ds.indexer.Search)(ctx, q)
}

func (ds *datastoreImpl) searchRawClusters(ctx context.Context, q *v1.Query) ([]*storage.Cluster, error) {
	results, err := ds.Search(ctx, q)
	if err != nil {
		return nil, err
	}
	clusters := make([]*storage.Cluster, 0, len(results))
	for _, result := range results {
		cluster, _, err := ds.storage.GetCluster(result.ID)
		if err != nil {
			return nil, errors.Wrapf(err, "retrieving cluster with id '%s'", result.ID)
		}
		// The result may not exist if the object was deleted after the search
		if cluster == nil {
			continue
		}
		clusters = append(clusters, cluster)
	}
	return clusters, nil
}

// GetCluster is a pass through function to the underlying storage.
func (ds *datastoreImpl) GetCluster(ctx context.Context, id string) (*storage.Cluster, bool, error) {
	if ok, err := clusterSAC.ReadAllowed(ctx, sac.ClusterScopeKey(id)); err != nil || !ok {
		return nil, false, err
	}

	return ds.storage.GetCluster(id)
}

// GetCluster is a pass through function to the underlying storage.
func (ds *datastoreImpl) GetClusters(ctx context.Context) ([]*storage.Cluster, error) {
	if ok, err := clusterSAC.ReadAllowed(ctx); err != nil {
		return nil, err
	} else if ok {
		return ds.storage.GetClusters()
	}

	return ds.searchRawClusters(ctx, search.EmptyQuery())
}

// GetCluster is a pass through function to the underlying storage.
func (ds *datastoreImpl) CountClusters(ctx context.Context) (int, error) {
	if ok, err := clusterSAC.ReadAllowed(ctx); err != nil {
		return 0, err
	} else if ok {
		return ds.storage.CountClusters()
	}

	visible, err := ds.Search(ctx, search.EmptyQuery())
	if err != nil {
		return 0, err
	}
	return len(visible), nil
}

// GetCluster is a pass through function to the underlying storage.
func (ds *datastoreImpl) AddCluster(ctx context.Context, cluster *storage.Cluster) (string, error) {
	if ok, err := clusterSAC.WriteAllowed(ctx, sac.ClusterScopeKey(cluster.GetId())); err != nil {
		return "", err
	} else if !ok {
		return "", errors.New("permission denied")
	}

	id, err := ds.storage.AddCluster(cluster)
	if err != nil {
		return "", err
	}
	return id, ds.indexer.AddCluster(cluster)
}

// GetCluster is a pass through function to the underlying storage.
func (ds *datastoreImpl) UpdateCluster(ctx context.Context, cluster *storage.Cluster) error {
	if ok, err := clusterSAC.WriteAllowed(ctx, sac.ClusterScopeKey(cluster.GetId())); err != nil {
		return err
	} else if !ok {
		return errors.New("permission denied")
	}

	if err := ds.storage.UpdateCluster(cluster); err != nil {
		return err
	}
	if err := ds.indexer.AddCluster(cluster); err != nil {
		return err
	}
	conn := ds.cm.GetConnection(cluster.GetId())
	if conn == nil {
		return nil
	}
	err := conn.InjectMessage(concurrency.Never(), &central.MsgToSensor{
		Msg: &central.MsgToSensor_ClusterConfig{
			ClusterConfig: &central.ClusterConfig{
				Config: cluster.GetDynamicConfig(),
			},
		},
	})
	if err != nil {
		// This is just logged because the connection could have been broken during the config send and we should handle it gracefully
		log.Error(err)
	}
	return nil
}

// GetCluster is a pass through function to the underlying storage.
func (ds *datastoreImpl) UpdateClusterContactTime(ctx context.Context, id string, t time.Time) error {
	if ok, err := clusterSAC.WriteAllowed(ctx, sac.ClusterScopeKey(id)); err != nil {
		return err
	} else if !ok {
		return errors.New("permission denied")
	}

	return ds.storage.UpdateClusterContactTime(id, t)
}

// RemoveCluster removes a cluster from the storage and the indexer
func (ds *datastoreImpl) RemoveCluster(ctx context.Context, id string, done *concurrency.Signal) error {
	if ok, err := clusterSAC.WriteAllowed(ctx, sac.ClusterScopeKey(id)); err != nil {
		return err
	} else if !ok {
		return errors.New("permission denied")
	}

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

	deleteRelatedCtx := sac.WithGlobalAccessScopeChecker(ctx,
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Deployment, resources.Alert, resources.Node, resources.Secret),
		))
	go ds.postRemoveCluster(deleteRelatedCtx, cluster, done)
	return ds.indexer.DeleteCluster(id)
}

func (ds *datastoreImpl) postRemoveCluster(ctx context.Context, cluster *storage.Cluster, done *concurrency.Signal) {
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
	deployments, err := ds.getDeployments(ctx, cluster)
	if err != nil {
		log.Errorf("failed to get deployments for removed cluster %s: %v", cluster.GetId(), err)
	}
	// Tombstone each deployment and mark alerts stale.
	for _, deployment := range deployments {
		alerts, err := ds.getAlerts(ctx, deployment)
		if err != nil {
			log.Errorf("failed to retrieve alerts for deployment %s: %v", deployment.GetId(), err)
		} else {
			err = ds.markAlertsStale(ctx, alerts)
			if err != nil {
				log.Errorf("failed to mark alerts for deployment %s as stale: %v", deployment.GetId(), err)
			}
		}

		err = ds.dds.RemoveDeployment(ctx, cluster.GetId(), deployment.GetId())
		if err != nil {
			log.Errorf("failed to remove deployment %s in deleted cluster: %v", deployment.GetId(), err)
		}
	}

	// Remove nodes associated with this cluster
	if err := ds.ns.RemoveClusterNodeStores(ctx, cluster.GetId()); err != nil {
		log.Errorf("failed to remove nodes for cluster %s: %v", cluster.GetId(), err)
	}

	secrets, err := ds.getSecrets(ctx, cluster)
	if err != nil {
		log.Errorf("failed to obtain secrets in deleted cluster %s: %v", cluster.GetId(), err)
	}
	for _, s := range secrets {
		// Best effort to remove. If the object doesn't exist, then that is okay
		_ = ds.ss.RemoveSecret(ctx, s.GetId())
	}
	if done != nil {
		done.Signal()
	}
}

// RemoveCluster removes an cluster from the storage and the indexer
func (ds *datastoreImpl) getSecrets(ctx context.Context, cluster *storage.Cluster) ([]*storage.ListSecret, error) {
	q := search.NewQueryBuilder().AddExactMatches(search.ClusterID, cluster.GetId()).ProtoQuery()
	return ds.ss.SearchListSecrets(ctx, q)
}

// RemoveCluster removes an cluster from the storage and the indexer
func (ds *datastoreImpl) getDeployments(ctx context.Context, cluster *storage.Cluster) ([]*storage.ListDeployment, error) {
	deployments, err := ds.dds.ListDeployments(ctx)
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
func (ds *datastoreImpl) getAlerts(ctx context.Context, deployment *storage.ListDeployment) ([]*storage.Alert, error) {
	qb := search.NewQueryBuilder().AddStrings(search.ViolationState, storage.ViolationState_ACTIVE.String()).AddExactMatches(search.DeploymentID, deployment.GetId())

	existingAlerts, err := ds.ads.SearchRawAlerts(ctx, qb.ProtoQuery())
	if err != nil {
		log.Errorf("unable to get alert: %s", err)
		return nil, err
	}
	return existingAlerts, nil
}

func (ds *datastoreImpl) markAlertsStale(ctx context.Context, alerts []*storage.Alert) error {
	errorList := errorhelpers.NewErrorList("unable to mark some alerts stale")
	for _, alert := range alerts {
		errorList.AddError(ds.ads.MarkAlertStale(ctx, alert.GetId()))
		if errorList.ToError() == nil {
			// run notifier for all the resolved alerts
			ds.notifier.ProcessAlert(alert)
		}
	}
	return errorList.ToError()
}

func (ds *datastoreImpl) cleanUpNodeStore(ctx context.Context) {
	if err := ds.doCleanUpNodeStore(ctx); err != nil {
		log.Errorf("Error cleaning up cluster node stores: %v", err)
	}
}

func (ds *datastoreImpl) doCleanUpNodeStore(ctx context.Context) error {
	clusterNodeStores, err := ds.ns.GetAllClusterNodeStores(ctx, false)
	if err != nil {
		return errors.Wrap(err, "retrieving per-cluster node stores")
	}

	if len(clusterNodeStores) == 0 {
		return nil
	}

	clusterIDsInNodeStore := set.NewStringSet()
	for clusterID := range clusterNodeStores {
		clusterIDsInNodeStore.Add(clusterID)
	}

	clusters, err := ds.GetClusters(ctx)
	if err != nil {
		return errors.Wrap(err, "retrieving clusters")
	}
	for _, cluster := range clusters {
		clusterIDsInNodeStore.Remove(cluster.GetId())
	}

	return ds.ns.RemoveClusterNodeStores(ctx, clusterIDsInNodeStore.AsSlice()...)
}
