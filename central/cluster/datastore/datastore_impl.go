package datastore

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	alertDataStore "github.com/stackrox/rox/central/alert/datastore"
	"github.com/stackrox/rox/central/cluster/datastore/internal/search"
	"github.com/stackrox/rox/central/cluster/index"
	"github.com/stackrox/rox/central/cluster/store"
	deploymentDataStore "github.com/stackrox/rox/central/deployment/datastore"
	namespaceDataStore "github.com/stackrox/rox/central/namespace/datastore"
	nodeDataStore "github.com/stackrox/rox/central/node/globaldatastore"
	notifierProcessor "github.com/stackrox/rox/central/notifier/processor"
	"github.com/stackrox/rox/central/ranking"
	"github.com/stackrox/rox/central/role/resources"
	secretDataStore "github.com/stackrox/rox/central/secret/datastore"
	"github.com/stackrox/rox/central/sensor/service/connection"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/sac"
	pkgSearch "github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/simplecache"
	"github.com/stackrox/rox/pkg/utils"
)

const (
	connectionTerminationTimeout = 5 * time.Second
)

var (
	clusterSAC = sac.ForResource(resources.Cluster)
)

type datastoreImpl struct {
	indexer  index.Indexer
	storage  store.Store
	notifier notifierProcessor.Processor
	searcher search.Searcher

	alertDataStore      alertDataStore.DataStore
	namespaceDataStore  namespaceDataStore.DataStore
	deploymentDataStore deploymentDataStore.DataStore
	nodeDataStore       nodeDataStore.GlobalDataStore
	secretsDataStore    secretDataStore.DataStore
	cm                  connection.Manager

	clusterRanker *ranking.Ranker

	cache simplecache.Cache
}

func (ds *datastoreImpl) UpdateClusterUpgradeStatus(ctx context.Context, id string, upgradeStatus *storage.ClusterUpgradeStatus) error {
	if err := checkWriteSac(ctx, id); err != nil {
		return err
	}
	return ds.storage.UpdateClusterUpgradeStatus(id, upgradeStatus)
}

func (ds *datastoreImpl) UpdateClusterStatus(ctx context.Context, id string, status *storage.ClusterStatus) error {
	if err := checkWriteSac(ctx, id); err != nil {
		return err
	}

	return ds.storage.UpdateClusterStatus(id, status)
}

func (ds *datastoreImpl) buildIndex() error {
	clusters, err := ds.storage.GetClusters()
	if err != nil {
		return err
	}
	for _, c := range clusters {
		ds.cache.Add(c.GetId(), c.GetName())
	}
	return ds.indexer.AddClusters(clusters)
}

func (ds *datastoreImpl) Search(ctx context.Context, q *v1.Query) ([]pkgSearch.Result, error) {
	return ds.searcher.Search(ctx, q)
}

func (ds *datastoreImpl) SearchResults(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error) {
	return ds.searcher.SearchResults(ctx, q)
}

func (ds *datastoreImpl) searchRawClusters(ctx context.Context, q *v1.Query) ([]*storage.Cluster, error) {
	clusters, err := ds.searcher.SearchClusters(ctx, q)
	if err != nil {
		return nil, err
	}

	ds.updateClusterPriority(clusters...)
	return clusters, nil
}

func (ds *datastoreImpl) GetCluster(ctx context.Context, id string) (*storage.Cluster, bool, error) {
	if ok, err := clusterSAC.ReadAllowed(ctx, sac.ClusterScopeKey(id)); err != nil || !ok {
		return nil, false, err
	}

	cluster, found, err := ds.storage.GetCluster(id)
	if err != nil || !found {
		return nil, false, err
	}

	ds.updateClusterPriority(cluster)
	return cluster, true, nil
}

func (ds *datastoreImpl) GetClusters(ctx context.Context) ([]*storage.Cluster, error) {
	if ok, err := clusterSAC.ReadAllowed(ctx); err != nil {
		return nil, err
	} else if ok {
		clusters, err := ds.storage.GetClusters()
		if err != nil {
			return nil, err
		}
		ds.updateClusterPriority(clusters...)
		return clusters, nil
	}

	return ds.searchRawClusters(ctx, pkgSearch.EmptyQuery())
}

func (ds *datastoreImpl) GetClusterName(ctx context.Context, id string) (string, bool, error) {
	if ok, err := clusterSAC.ReadAllowed(ctx, sac.ClusterScopeKey(id)); err != nil || !ok {
		return "", false, err
	}
	val, ok := ds.cache.Get(id)
	if !ok {
		return "", false, nil
	}
	return val.(string), true, nil
}

func (ds *datastoreImpl) Exists(ctx context.Context, id string) (bool, error) {
	if ok, err := clusterSAC.ReadAllowed(ctx, sac.ClusterScopeKey(id)); err != nil || !ok {
		return false, err
	}
	_, ok := ds.cache.Get(id)
	return ok, nil
}

func (ds *datastoreImpl) SearchRawClusters(ctx context.Context, q *v1.Query) ([]*storage.Cluster, error) {
	return ds.searchRawClusters(ctx, q)
}

func (ds *datastoreImpl) CountClusters(ctx context.Context) (int, error) {
	if ok, err := clusterSAC.ReadAllowed(ctx); err != nil {
		return 0, err
	} else if ok {
		return ds.storage.CountClusters()
	}

	visible, err := ds.Search(ctx, pkgSearch.EmptyQuery())
	if err != nil {
		return 0, err
	}
	return len(visible), nil
}

func checkWriteSac(ctx context.Context, clusterID string) error {
	if ok, err := clusterSAC.WriteAllowed(ctx, sac.ClusterScopeKey(clusterID)); err != nil {
		return err
	} else if !ok {
		return sac.ErrPermissionDenied
	}
	return nil
}

func (ds *datastoreImpl) AddCluster(ctx context.Context, cluster *storage.Cluster) (string, error) {
	if err := checkWriteSac(ctx, cluster.GetId()); err != nil {
		return "", err
	}

	id, err := ds.storage.AddCluster(cluster)
	if err != nil {
		return "", err
	}
	if err := ds.indexer.AddCluster(cluster); err != nil {
		return "", err
	}
	ds.cache.Add(cluster.GetId(), cluster.GetName())
	return id, nil
}

func (ds *datastoreImpl) UpdateCluster(ctx context.Context, cluster *storage.Cluster) error {
	if err := checkWriteSac(ctx, cluster.GetId()); err != nil {
		return err
	}

	if err := ds.storage.UpdateCluster(cluster); err != nil {
		return err
	}
	if err := ds.indexer.AddCluster(cluster); err != nil {
		return err
	}

	ds.cache.Add(cluster.GetId(), cluster.GetName())

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

func (ds *datastoreImpl) UpdateClusterContactTimes(ctx context.Context, t time.Time, ids ...string) error {
	if ok, err := clusterSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return errors.New("permission denied")
	}

	return ds.storage.UpdateClusterContactTimes(t, ids...)
}

func (ds *datastoreImpl) RemoveCluster(ctx context.Context, id string, done *concurrency.Signal) error {
	if err := checkWriteSac(ctx, id); err != nil {
		return err
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
	ds.cache.Remove(id)

	deleteRelatedCtx := sac.WithGlobalAccessScopeChecker(ctx,
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Namespace, resources.Deployment, resources.Alert, resources.Node, resources.Secret),
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
				utils.Should(errors.Errorf("connection to sensor from cluster %s not terminated after %v", cluster.GetId(), connectionTerminationTimeout))
			}
		}
	}

	// Remove ranker record here since removal is not handled in risk store as no entry present for cluster
	ds.clusterRanker.Remove(cluster.GetId())

	ds.removeClusterNamespaces(ctx, cluster)

	// Tombstone each deployment and mark alerts stale.
	ds.removeClusterDeployments(ctx, cluster)

	// Remove nodes associated with this cluster
	if err := ds.nodeDataStore.RemoveClusterNodeStores(ctx, cluster.GetId()); err != nil {
		log.Errorf("failed to remove nodes for cluster %s: %v", cluster.GetId(), err)
	}

	ds.removeClusterSecrets(ctx, cluster)
	if done != nil {
		done.Signal()
	}
}

func (ds *datastoreImpl) removeClusterNamespaces(ctx context.Context, cluster *storage.Cluster) {
	q := pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.ClusterID, cluster.GetId()).ProtoQuery()
	namespaces, err := ds.namespaceDataStore.Search(ctx, q)
	if err != nil {
		log.Errorf("failed to get namespaces for removed cluster %s: %v", cluster.GetId(), err)
	}

	for _, namespace := range namespaces {
		err = ds.namespaceDataStore.RemoveNamespace(ctx, namespace.ID)
		if err != nil {
			log.Errorf("failed to remove namespace %s in deleted cluster: %v", namespace.ID, err)
		}
	}

}

func (ds *datastoreImpl) removeClusterDeployments(ctx context.Context, cluster *storage.Cluster) {
	q := pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.ClusterID, cluster.GetId()).ProtoQuery()
	deployments, err := ds.deploymentDataStore.Search(ctx, q)
	if err != nil {
		log.Errorf("failed to get deployments for removed cluster %s: %v", cluster.GetId(), err)
	}
	// Tombstone each deployment and mark alerts stale.
	for _, deployment := range deployments {
		alerts, err := ds.getAlerts(ctx, deployment.ID)
		if err != nil {
			log.Errorf("failed to retrieve alerts for deployment %s: %v", deployment.ID, err)
		} else {
			err = ds.markAlertsStale(ctx, alerts)
			if err != nil {
				log.Errorf("failed to mark alerts for deployment %s as stale: %v", deployment.ID, err)
			}
		}

		err = ds.deploymentDataStore.RemoveDeployment(ctx, cluster.GetId(), deployment.ID)
		if err != nil {
			log.Errorf("failed to remove deployment %s in deleted cluster: %v", deployment.ID, err)
		}
	}

}

func (ds *datastoreImpl) removeClusterSecrets(ctx context.Context, cluster *storage.Cluster) {
	secrets, err := ds.getSecrets(ctx, cluster)
	if err != nil {
		log.Errorf("failed to obtain secrets in deleted cluster %s: %v", cluster.GetId(), err)
	}
	for _, s := range secrets {
		// Best effort to remove. If the object doesn't exist, then that is okay
		_ = ds.secretsDataStore.RemoveSecret(ctx, s.GetId())
	}
}

func (ds *datastoreImpl) getSecrets(ctx context.Context, cluster *storage.Cluster) ([]*storage.ListSecret, error) {
	q := pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.ClusterID, cluster.GetId()).ProtoQuery()
	return ds.secretsDataStore.SearchListSecrets(ctx, q)
}

func (ds *datastoreImpl) getAlerts(ctx context.Context, deploymentID string) ([]*storage.Alert, error) {
	q := pkgSearch.NewQueryBuilder().
		AddStrings(pkgSearch.ViolationState, storage.ViolationState_ACTIVE.String()).
		AddExactMatches(pkgSearch.DeploymentID, deploymentID).ProtoQuery()
	return ds.alertDataStore.SearchRawAlerts(ctx, q)
}

func (ds *datastoreImpl) markAlertsStale(ctx context.Context, alerts []*storage.Alert) error {
	errorList := errorhelpers.NewErrorList("unable to mark some alerts stale")
	for _, alert := range alerts {
		errorList.AddError(ds.alertDataStore.MarkAlertStale(ctx, alert.GetId()))
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
	clusterNodeStores, err := ds.nodeDataStore.GetAllClusterNodeStores(ctx, false)
	if err != nil {
		return errors.Wrap(err, "retrieving per-cluster node stores")
	}

	if len(clusterNodeStores) == 0 {
		return nil
	}

	orphanedClusterIDsInNodeStore := set.NewStringSet()
	for clusterID := range clusterNodeStores {
		orphanedClusterIDsInNodeStore.Add(clusterID)
	}

	clusters, err := ds.GetClusters(ctx)
	if err != nil {
		return errors.Wrap(err, "retrieving clusters")
	}
	for _, cluster := range clusters {
		orphanedClusterIDsInNodeStore.Remove(cluster.GetId())
	}

	return ds.nodeDataStore.RemoveClusterNodeStores(ctx, orphanedClusterIDsInNodeStore.AsSlice()...)
}

func (ds *datastoreImpl) updateClusterPriority(clusters ...*storage.Cluster) {
	for _, cluster := range clusters {
		cluster.Priority = ds.clusterRanker.GetRankForID(cluster.GetId())
	}
}
