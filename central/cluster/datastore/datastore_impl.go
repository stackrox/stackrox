package datastore

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	alertDataStore "github.com/stackrox/rox/central/alert/datastore"
	"github.com/stackrox/rox/central/cluster/datastore/internal/search"
	clusterStore "github.com/stackrox/rox/central/cluster/store/cluster"
	clusterHealthStore "github.com/stackrox/rox/central/cluster/store/clusterhealth"
	clusterCVEDS "github.com/stackrox/rox/central/cve/cluster/datastore"
	deploymentDataStore "github.com/stackrox/rox/central/deployment/datastore"
	imageIntegrationDataStore "github.com/stackrox/rox/central/imageintegration/datastore"
	namespaceDataStore "github.com/stackrox/rox/central/namespace/datastore"
	networkBaselineManager "github.com/stackrox/rox/central/networkbaseline/manager"
	netEntityDataStore "github.com/stackrox/rox/central/networkgraph/entity/datastore"
	netFlowDataStore "github.com/stackrox/rox/central/networkgraph/flow/datastore"
	nodeDataStore "github.com/stackrox/rox/central/node/datastore"
	podDataStore "github.com/stackrox/rox/central/pod/datastore"
	"github.com/stackrox/rox/central/ranking"
	roleDataStore "github.com/stackrox/rox/central/rbac/k8srole/datastore"
	roleBindingDataStore "github.com/stackrox/rox/central/rbac/k8srolebinding/datastore"
	secretDataStore "github.com/stackrox/rox/central/secret/datastore"
	"github.com/stackrox/rox/central/sensor/service/common"
	"github.com/stackrox/rox/central/sensor/service/connection"
	serviceAccountDataStore "github.com/stackrox/rox/central/serviceaccount/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/cache/objectarraycache"
	clusterValidation "github.com/stackrox/rox/pkg/cluster"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/images/defaults"
	notifierProcessor "github.com/stackrox/rox/pkg/notifier"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/effectiveaccessscope"
	"github.com/stackrox/rox/pkg/sac/resources"
	pkgSearch "github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/simplecache"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/uuid"
)

const (
	// clusterMoveGracePeriod determines the amount of time that has to pass before a (logical) StackRox cluster can
	// be moved to a different (physical) Kubernetes cluster.
	clusterMoveGracePeriod = 3 * time.Minute
)

const (
	defaultAdmissionControllerTimeout = 3

	cacheRefreshPeriod = 5 * time.Second
)

var (
	clusterSAC = sac.ForResource(resources.Cluster)
)

type datastoreImpl struct {
	clusterStorage            clusterStore.Store
	clusterHealthStorage      clusterHealthStore.Store
	clusterCVEDataStore       clusterCVEDS.DataStore
	alertDataStore            alertDataStore.DataStore
	imageIntegrationDataStore imageIntegrationDataStore.DataStore
	namespaceDataStore        namespaceDataStore.DataStore
	deploymentDataStore       deploymentDataStore.DataStore
	nodeDataStore             nodeDataStore.DataStore
	podDataStore              podDataStore.DataStore
	secretsDataStore          secretDataStore.DataStore
	netFlowsDataStore         netFlowDataStore.ClusterDataStore
	netEntityDataStore        netEntityDataStore.EntityDataStore
	serviceAccountDataStore   serviceAccountDataStore.DataStore
	roleDataStore             roleDataStore.DataStore
	roleBindingDataStore      roleBindingDataStore.DataStore
	cm                        connection.Manager
	networkBaselineMgr        networkBaselineManager.Manager

	notifier      notifierProcessor.Processor
	clusterRanker *ranking.Ranker

	idToNameCache simplecache.Cache
	nameToIDCache simplecache.Cache

	searcher search.Searcher

	objectCacheForSAC *objectarraycache.ObjectArrayCache[effectiveaccessscope.ClusterForSAC]

	lock sync.Mutex
}

func (ds *datastoreImpl) UpdateClusterUpgradeStatus(ctx context.Context, id string, upgradeStatus *storage.ClusterUpgradeStatus) error {
	if err := checkWriteSac(ctx, id); err != nil {
		return err
	}

	ds.lock.Lock()
	defer ds.lock.Unlock()

	cluster, err := ds.getClusterOnly(ctx, id)
	if err != nil {
		return err
	}

	if cluster.GetStatus() == nil {
		cluster.Status = &storage.ClusterStatus{}
	}

	cluster.Status.UpgradeStatus = upgradeStatus
	return ds.clusterStorage.Upsert(ctx, cluster)
}

func (ds *datastoreImpl) UpdateClusterCertExpiryStatus(ctx context.Context, id string, clusterCertExpiryStatus *storage.ClusterCertExpiryStatus) error {
	if err := checkWriteSac(ctx, id); err != nil {
		return err
	}

	ds.lock.Lock()
	defer ds.lock.Unlock()

	cluster, err := ds.getClusterOnly(ctx, id)
	if err != nil {
		return err
	}

	if cluster.GetStatus() == nil {
		cluster.Status = &storage.ClusterStatus{}
	}

	cluster.Status.CertExpiryStatus = clusterCertExpiryStatus
	return ds.clusterStorage.Upsert(ctx, cluster)
}

func (ds *datastoreImpl) UpdateClusterStatus(ctx context.Context, id string, status *storage.ClusterStatus) error {
	if err := checkWriteSac(ctx, id); err != nil {
		return err
	}

	cluster, err := ds.getClusterOnly(ctx, id)
	if err != nil {
		return err
	}

	status.UpgradeStatus = cluster.GetStatus().GetUpgradeStatus()
	status.CertExpiryStatus = cluster.GetStatus().GetCertExpiryStatus()
	cluster.Status = status

	return ds.clusterStorage.Upsert(ctx, cluster)
}

func (ds *datastoreImpl) buildCache(ctx context.Context) error {
	clusters, err := ds.collectClusters(ctx)
	if err != nil {
		return err
	}

	clusterHealthStatuses := make(map[string]*storage.ClusterHealthStatus)
	walkFn := func() error {
		clusterHealthStatuses = make(map[string]*storage.ClusterHealthStatus)
		return ds.clusterHealthStorage.Walk(ctx, func(healthInfo *storage.ClusterHealthStatus) error {
			clusterHealthStatuses[healthInfo.Id] = healthInfo
			return nil
		})
	}
	if err := pgutils.RetryIfPostgres(walkFn); err != nil {
		return err
	}

	for _, c := range clusters {
		ds.idToNameCache.Add(c.GetId(), c.GetName())
		ds.nameToIDCache.Add(c.GetName(), c.GetId())
		c.HealthStatus = clusterHealthStatuses[c.GetId()]
	}
	return nil
}

func (ds *datastoreImpl) registerClusterForNetworkGraphExtSrcs() error {
	ctx := sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Node, resources.NetworkGraph)))

	clusters, err := ds.collectClusters(ctx)
	if err != nil {
		return err
	}
	for _, cluster := range clusters {
		ds.netEntityDataStore.RegisterCluster(ctx, cluster.GetId())
	}
	return nil
}

func (ds *datastoreImpl) Search(ctx context.Context, q *v1.Query) ([]pkgSearch.Result, error) {
	return ds.searcher.Search(ctx, q)
}

// Count returns the number of search results from the query
func (ds *datastoreImpl) Count(ctx context.Context, q *v1.Query) (int, error) {
	return ds.searcher.Count(ctx, q)
}

func (ds *datastoreImpl) SearchResults(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error) {
	return ds.searcher.SearchResults(ctx, q)
}

func (ds *datastoreImpl) searchRawClusters(ctx context.Context, q *v1.Query) ([]*storage.Cluster, error) {
	clusters, err := ds.searcher.SearchClusters(ctx, q)
	if err != nil {
		return nil, err
	}

	ds.populateHealthInfos(ctx, clusters...)
	ds.updateClusterPriority(clusters...)
	return clusters, nil
}

func (ds *datastoreImpl) GetCluster(ctx context.Context, id string) (*storage.Cluster, bool, error) {
	cluster, found, err := ds.clusterStorage.Get(ctx, id)
	if err != nil || !found {
		return nil, false, err
	}
	if ok, err := clusterSAC.ReadAllowed(ctx, sac.ClusterScopeKey(id)); err != nil || !ok {
		return nil, false, err
	}

	ds.populateHealthInfos(ctx, cluster)
	ds.updateClusterPriority(cluster)
	return cluster, true, nil
}

func (ds *datastoreImpl) GetClusters(ctx context.Context) ([]*storage.Cluster, error) {
	ok, err := clusterSAC.ReadAllowed(ctx)
	if err != nil {
		return nil, err
	} else if ok {
		clusters, err := ds.collectClusters(ctx)
		if err != nil {
			return nil, err
		}

		ds.populateHealthInfos(ctx, clusters...)
		ds.updateClusterPriority(clusters...)
		return clusters, nil
	}

	return ds.searchRawClusters(ctx, pkgSearch.EmptyQuery())
}

func (ds *datastoreImpl) GetClustersForSAC(ctx context.Context) ([]effectiveaccessscope.ClusterForSAC, error) {
	return ds.objectCacheForSAC.GetObjects(ctx)
}

func (ds *datastoreImpl) getClustersForSAC(ctx context.Context) ([]effectiveaccessscope.ClusterForSAC, error) {
	_, err := clusterSAC.ReadAllowed(ctx)
	if err != nil {
		return nil, err
	}

	clusters := make([]effectiveaccessscope.ClusterForSAC, 0)
	err = ds.clusterStorage.Walk(ctx, func(obj *storage.Cluster) error {
		clusters = append(clusters, effectiveaccessscope.StorageClusterToClusterForSAC(obj))
		return nil
	})
	if err != nil {
		return nil, err
	}
	return clusters, nil
}

func (ds *datastoreImpl) GetClusterName(ctx context.Context, id string) (string, bool, error) {
	if ok, err := clusterSAC.ReadAllowed(ctx, sac.ClusterScopeKey(id)); err != nil || !ok {
		return "", false, err
	}
	val, ok := ds.idToNameCache.Get(id)
	if !ok {
		return "", false, nil
	}
	return val.(string), true, nil
}

func (ds *datastoreImpl) Exists(ctx context.Context, id string) (bool, error) {
	if ok, err := clusterSAC.ReadAllowed(ctx, sac.ClusterScopeKey(id)); err != nil || !ok {
		return false, err
	}
	_, ok := ds.idToNameCache.Get(id)
	return ok, nil
}

func (ds *datastoreImpl) SearchRawClusters(ctx context.Context, q *v1.Query) ([]*storage.Cluster, error) {
	clusters, err := ds.searchRawClusters(ctx, q)
	if err != nil {
		return nil, err
	}
	return clusters, nil
}

func (ds *datastoreImpl) CountClusters(ctx context.Context) (int, error) {
	if ok, err := clusterSAC.ReadAllowed(ctx); err != nil {
		return 0, err
	} else if ok {
		return ds.clusterStorage.Count(ctx)
	}

	return ds.Count(ctx, pkgSearch.EmptyQuery())
}

func checkWriteSac(ctx context.Context, clusterID string) error {
	if ok, err := clusterSAC.WriteAllowed(ctx, sac.ClusterScopeKey(clusterID)); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}
	return nil
}

func (ds *datastoreImpl) AddCluster(ctx context.Context, cluster *storage.Cluster) (string, error) {
	if err := checkWriteSac(ctx, cluster.GetId()); err != nil {
		return "", err
	}
	_, found := ds.nameToIDCache.Get(cluster.Name)
	if found {
		return "", errox.AlreadyExists.Newf("the cluster with name %s exists, cannot re-add it", cluster.GetName())
	}

	ds.lock.Lock()
	defer ds.lock.Unlock()

	return ds.addClusterNoLock(ctx, cluster)
}

func (ds *datastoreImpl) addClusterNoLock(ctx context.Context, cluster *storage.Cluster) (string, error) {
	if cluster.GetId() != "" {
		return "", errors.Errorf("cannot add a cluster that has already been assigned an id: %q", cluster.GetId())
	}

	if cluster.GetName() == "" {
		return "", errors.New("cannot add a cluster without name")
	}

	cluster.Id = uuid.NewV4().String()
	if err := ds.updateClusterNoLock(ctx, cluster); err != nil {
		return "", err
	}

	trackClusterRegistered(ctx, cluster)

	// Temporarily elevate permissions to create network flow store for the cluster.
	networkGraphElevatedCtx := sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.NetworkGraph)))

	if _, err := ds.netFlowsDataStore.CreateFlowStore(networkGraphElevatedCtx, cluster.GetId()); err != nil {
		return "", errors.Wrapf(err, "could not create flow store for cluster %s", cluster.GetId())
	}
	return cluster.GetId(), nil
}

func (ds *datastoreImpl) UpdateCluster(ctx context.Context, cluster *storage.Cluster) error {
	if err := checkWriteSac(ctx, cluster.GetId()); err != nil {
		return err
	}

	ds.lock.Lock()
	defer ds.lock.Unlock()

	existingCluster, exists, err := ds.clusterStorage.Get(ctx, cluster.GetId())
	if err != nil {
		return err
	}
	if exists {
		if cluster.GetName() != existingCluster.GetName() {
			return errors.Errorf("cannot update cluster. Cluster name change from %s not permitted", existingCluster.GetName())
		}
		if cluster.GetManagedBy() != existingCluster.GetManagedBy() {
			return errors.Errorf("Cannot update cluster. Cluster manager type change from %s not permitted.", existingCluster.GetManagedBy())
		}
		cluster.Status = existingCluster.GetStatus()
	}

	if err := ds.updateClusterNoLock(ctx, cluster); err != nil {
		return err
	}

	conn := ds.cm.GetConnection(cluster.GetId())
	if conn == nil {
		return nil
	}
	err = conn.InjectMessage(concurrency.Never(), &central.MsgToSensor{
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

func (ds *datastoreImpl) UpdateClusterHealth(ctx context.Context, id string, clusterHealthStatus *storage.ClusterHealthStatus) error {
	if id == "" {
		return errors.New("cannot update cluster health. cluster id not provided")
	}

	if clusterHealthStatus == nil {
		return errors.Errorf("cannot update health for cluster %s. No health information available", id)
	}

	if err := checkWriteSac(ctx, id); err != nil {
		return err
	}

	ds.lock.Lock()
	defer ds.lock.Unlock()

	oldHealth, _, err := ds.clusterHealthStorage.Get(ctx, id)
	if err != nil {
		return err
	}

	clusterHealthStatus.Id = id
	if err := ds.clusterHealthStorage.Upsert(ctx, clusterHealthStatus); err != nil {
		return err
	}

	// If no change in cluster health status, no need to rebuild index
	if clusterHealthStatus.GetSensorHealthStatus() == oldHealth.GetSensorHealthStatus() && clusterHealthStatus.GetCollectorHealthStatus() == oldHealth.GetCollectorHealthStatus() {
		return nil
	}

	cluster, exists, err := ds.clusterStorage.Get(ctx, id)
	if err != nil {
		return err
	}

	if !exists {
		return nil
	}
	cluster.HealthStatus = clusterHealthStatus

	if oldHealth.GetSensorHealthStatus() == storage.ClusterHealthStatus_UNINITIALIZED &&
		clusterHealthStatus.GetSensorHealthStatus() != storage.ClusterHealthStatus_UNINITIALIZED {
		trackClusterInitialized(cluster)
	}
	return nil
}

func (ds *datastoreImpl) UpdateSensorDeploymentIdentification(ctx context.Context, id string, identification *storage.SensorDeploymentIdentification) error {
	if err := checkWriteSac(ctx, id); err != nil {
		return err
	}

	ds.lock.Lock()
	defer ds.lock.Unlock()

	cluster, err := ds.getClusterOnly(ctx, id)
	if err != nil {
		return err
	}

	cluster.MostRecentSensorId = identification
	return ds.clusterStorage.Upsert(ctx, cluster)
}

func (ds *datastoreImpl) UpdateAuditLogFileStates(ctx context.Context, id string, states map[string]*storage.AuditLogFileState) error {
	if id == "" {
		return errors.New("cannot update audit log file states because cluster id was not provided")
	}
	if len(states) == 0 {
		return errors.Errorf("cannot update audit log file states for cluster %s. No state information available", id)
	}

	if err := checkWriteSac(ctx, id); err != nil {
		return err
	}

	ds.lock.Lock()
	defer ds.lock.Unlock()

	cluster, err := ds.getClusterOnly(ctx, id)
	if err != nil {
		return err
	}

	// If a state is missing in the new update, keep it in the saved state.
	// It could be that compliance is down temporarily and we don't want to lose the data
	if cluster.GetAuditLogState() == nil {
		cluster.AuditLogState = make(map[string]*storage.AuditLogFileState)
	}
	for node, state := range states {
		cluster.AuditLogState[node] = state
	}

	return ds.clusterStorage.Upsert(ctx, cluster)
}

func (ds *datastoreImpl) RemoveCluster(ctx context.Context, id string, done *concurrency.Signal) error {
	if err := checkWriteSac(ctx, id); err != nil {
		return err
	}

	ds.lock.Lock()
	defer ds.lock.Unlock()

	// Fetch the cluster an confirm it exists.
	cluster, exists, err := ds.clusterStorage.Get(ctx, id)
	if !exists {
		return errors.Errorf("unable to find cluster %q", id)
	}
	if err != nil {
		return err
	}

	if err := ds.clusterStorage.Delete(ctx, id); err != nil {
		return errors.Wrapf(err, "failed to remove cluster %q", id)
	}
	ds.idToNameCache.Remove(id)
	ds.nameToIDCache.Remove(cluster.GetName())

	deleteRelatedCtx := sac.WithAllAccess(context.Background())
	go ds.postRemoveCluster(deleteRelatedCtx, cluster, done)
	return nil
}

func (ds *datastoreImpl) postRemoveCluster(ctx context.Context, cluster *storage.Cluster, done *concurrency.Signal) {
	// Terminate the cluster connection to prevent new data from being stored.
	if ds.cm != nil {
		ds.cm.CloseConnection(cluster.GetId())
	}
	ds.removeClusterImageIntegrations(ctx, cluster)

	// Remove ranker record here since removal is not handled in risk store as no entry present for cluster
	ds.clusterRanker.Remove(cluster.GetId())

	ds.removeClusterNamespaces(ctx, cluster)

	// Tombstone each deployment and mark alerts stale.
	removedDeployments := ds.removeClusterDeployments(ctx, cluster)

	ds.removeClusterPods(ctx, cluster)

	// Remove nodes associated with this cluster
	if err := ds.nodeDataStore.DeleteAllNodesForCluster(ctx, cluster.GetId()); err != nil {
		log.Errorf("failed to remove nodes for cluster %s: %v", cluster.GetId(), err)
	}

	if err := ds.netEntityDataStore.DeleteExternalNetworkEntitiesForCluster(ctx, cluster.GetId()); err != nil {
		log.Errorf("failed to delete external network graph entities for removed cluster %s: %v", cluster.GetId(), err)
	}

	err := ds.networkBaselineMgr.ProcessPostClusterDelete(removedDeployments)
	if err != nil {
		log.Errorf("failed to delete network baselines associated with this cluster %q: %v", cluster.GetId(), err)
	}

	ds.removeClusterSecrets(ctx, cluster)
	ds.removeClusterServiceAccounts(ctx, cluster)
	ds.removeK8SRoles(ctx, cluster)
	ds.removeRoleBindings(ctx, cluster)

	if err := ds.clusterCVEDataStore.DeleteClusterCVEsInternal(ctx, cluster.GetId()); err != nil {
		log.Errorf("Failed to delete cluster cves for cluster %q: %v ", cluster.GetId(), err)
	}

	if done != nil {
		done.Signal()
	}
}

func (ds *datastoreImpl) removeClusterImageIntegrations(ctx context.Context, cluster *storage.Cluster) {
	q := pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.ClusterID, cluster.GetId()).ProtoQuery()

	imageIntegrations, err := ds.imageIntegrationDataStore.Search(ctx, q)
	if err != nil {
		log.Errorf("failed to get image integrations for removed cluster %s: %v", cluster.GetId(), err)
		return
	}
	for _, imageIntegration := range imageIntegrations {
		err = ds.imageIntegrationDataStore.RemoveImageIntegration(ctx, imageIntegration.ID)
		if err != nil {
			log.Errorf("failed to remove image integration %s in deleted cluster: %v", imageIntegration.ID, err)
		}
	}
}

func (ds *datastoreImpl) removeClusterNamespaces(ctx context.Context, cluster *storage.Cluster) {
	q := pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.ClusterID, cluster.GetId()).ProtoQuery()
	namespaces, err := ds.namespaceDataStore.Search(ctx, q)
	if err != nil {
		log.Errorf("Failed to get namespaces for removed cluster %s: %v", cluster.GetId(), err)
	}

	for _, namespace := range namespaces {
		err = ds.namespaceDataStore.RemoveNamespace(ctx, namespace.ID)
		if err != nil {
			log.Errorf("Failed to remove namespace %s in deleted cluster: %v", namespace.ID, err)
		}
	}

}

func (ds *datastoreImpl) removeClusterPods(ctx context.Context, cluster *storage.Cluster) {
	q := pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.ClusterID, cluster.GetId()).ProtoQuery()
	pods, err := ds.podDataStore.Search(ctx, q)
	if err != nil {
		log.Errorf("Failed to get pods for removed cluster %s: %v", cluster.GetId(), err)
		return
	}
	for _, pod := range pods {
		if err := ds.podDataStore.RemovePod(ctx, pod.ID); err != nil {
			log.Errorf("Failed to remove pod with id %s as part of removal of cluster %s: %v", pod.ID, cluster.GetId(), err)
		}
	}
}

func (ds *datastoreImpl) removeClusterDeployments(ctx context.Context, cluster *storage.Cluster) []string {
	q := pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.ClusterID, cluster.GetId()).ProtoQuery()
	deployments, err := ds.deploymentDataStore.Search(ctx, q)
	if err != nil {
		log.Errorf("failed to get deployments for removed cluster %s: %v", cluster.GetId(), err)
	}

	// Deployment IDs being removed.
	removedIDs := make([]string, 0, len(deployments))

	// Tombstone each deployment and mark alerts stale.
	for _, deployment := range deployments {
		alerts, err := ds.getAlerts(ctx, deployment.ID)
		if err != nil {
			log.Errorf("Failed to retrieve alerts for deployment %s: %v", deployment.ID, err)
		} else {
			err = ds.markAlertsStale(ctx, alerts)
			if err != nil {
				log.Errorf("Failed to mark alerts for deployment %s as stale: %v", deployment.ID, err)
			}
		}

		removedIDs = append(removedIDs, deployment.ID)
		err = ds.deploymentDataStore.RemoveDeployment(ctx, cluster.GetId(), deployment.ID)
		if err != nil {
			log.Errorf("Failed to remove deployment %s in deleted cluster: %v", deployment.ID, err)
		}
	}

	return removedIDs
}

func (ds *datastoreImpl) removeClusterSecrets(ctx context.Context, cluster *storage.Cluster) {
	secrets, err := ds.getSecrets(ctx, cluster)
	if err != nil {
		log.Errorf("Failed to obtain secrets in deleted cluster %s: %v", cluster.GetId(), err)
	}
	for _, s := range secrets {
		// Best effort to remove. If the object doesn't exist, then that is okay
		if err := ds.secretsDataStore.RemoveSecret(ctx, s.GetId()); err != nil {
			log.Errorf("Failed to remove secret with id %s from deleted cluster %s: %v", s.GetId(), cluster.GetId(), err)
		}
	}
}

func (ds *datastoreImpl) removeClusterServiceAccounts(ctx context.Context, cluster *storage.Cluster) {
	serviceAccounts, err := ds.getServiceAccounts(ctx, cluster)
	if err != nil {
		log.Errorf("Failed to find service accounts in deleted cluster %s: %v", cluster.GetId(), err)
	}
	for _, s := range serviceAccounts {
		// Best effort to remove. If the object doesn't exist, then that is okay
		if err := ds.serviceAccountDataStore.RemoveServiceAccount(ctx, s); err != nil {
			log.Errorf("Failed to remove service account with id %s from deleted cluster %s: %v", s, cluster.GetId(), err)
		}
	}
}

func (ds *datastoreImpl) removeK8SRoles(ctx context.Context, cluster *storage.Cluster) {
	roles, err := ds.getRoles(ctx, cluster)
	if err != nil {
		log.Errorf("Failed to find K8S roles in deleted cluster %s: %v", cluster.GetId(), err)
	}
	for _, r := range roles {
		// Best effort to remove. If the object doesn't exist, then that is okay
		if err := ds.roleDataStore.RemoveRole(ctx, r); err != nil {
			log.Errorf("failed to remove K8S role with id %s from deleted cluster %s: %v", r, cluster.GetId(), err)
		}
	}
}

func (ds *datastoreImpl) removeRoleBindings(ctx context.Context, cluster *storage.Cluster) {
	bindings, err := ds.getRoleBindings(ctx, cluster)
	if err != nil {
		log.Errorf("Failed to find K8S role bindings in deleted cluster %s: %v", cluster.GetId(), err)
	}
	for _, b := range bindings {
		// Best effort to remove. If the object doesn't exist, then that is okay
		if err := ds.roleBindingDataStore.RemoveRoleBinding(ctx, b); err != nil {
			log.Errorf("Failed to remove K8S role binding with id %s from deleted cluster %s: %v", b, cluster.GetId(), err)
		}
	}
}

func (ds *datastoreImpl) getRoleBindings(ctx context.Context, cluster *storage.Cluster) ([]string, error) {
	q := pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.ClusterID, cluster.GetId()).ProtoQuery()
	return convertSearchResultsToIDs(ds.roleBindingDataStore.Search(ctx, q))
}

func (ds *datastoreImpl) getRoles(ctx context.Context, cluster *storage.Cluster) ([]string, error) {
	q := pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.ClusterID, cluster.GetId()).ProtoQuery()
	return convertSearchResultsToIDs(ds.roleDataStore.Search(ctx, q))
}

func (ds *datastoreImpl) getServiceAccounts(ctx context.Context, cluster *storage.Cluster) ([]string, error) {
	q := pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.ClusterID, cluster.GetId()).ProtoQuery()
	return convertSearchResultsToIDs(ds.serviceAccountDataStore.Search(ctx, q))
}

func convertSearchResultsToIDs(res []pkgSearch.Result, err error) ([]string, error) {
	if err != nil {
		return nil, err
	}

	return pkgSearch.ResultsToIDs(res), nil
}

func (ds *datastoreImpl) getSecrets(ctx context.Context, cluster *storage.Cluster) ([]*storage.ListSecret, error) {
	q := pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.ClusterID, cluster.GetId()).ProtoQuery()
	return ds.secretsDataStore.SearchListSecrets(ctx, q)
}

func (ds *datastoreImpl) getAlerts(ctx context.Context, deploymentID string) ([]*storage.Alert, error) {
	q := pkgSearch.NewQueryBuilder().
		AddExactMatches(pkgSearch.ViolationState, storage.ViolationState_ACTIVE.String()).
		AddExactMatches(pkgSearch.DeploymentID, deploymentID).ProtoQuery()
	return ds.alertDataStore.SearchRawAlerts(ctx, q)
}

func (ds *datastoreImpl) markAlertsStale(ctx context.Context, alerts []*storage.Alert) error {
	if len(alerts) == 0 {
		return nil
	}

	ids := make([]string, 0, len(alerts))
	for _, alert := range alerts {
		ids = append(ids, alert.GetId())
	}
	resolvedAlerts, err := ds.alertDataStore.MarkAlertsResolvedBatch(ctx, ids...)
	if err != nil {
		return err
	}
	for _, resolvedAlert := range resolvedAlerts {
		ds.notifier.ProcessAlert(ctx, resolvedAlert)
	}
	return nil
}

func (ds *datastoreImpl) updateClusterPriority(clusters ...*storage.Cluster) {
	for _, cluster := range clusters {
		cluster.Priority = ds.clusterRanker.GetRankForID(cluster.GetId())
	}
}

func (ds *datastoreImpl) getClusterOnly(ctx context.Context, id string) (*storage.Cluster, error) {
	cluster, exists, err := ds.clusterStorage.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.Errorf("cluster %s not found", id)
	}
	return cluster, nil
}

func (ds *datastoreImpl) populateHealthInfos(ctx context.Context, clusters ...*storage.Cluster) {
	ids := make([]string, 0, len(clusters))
	for _, cluster := range clusters {
		ids = append(ids, cluster.GetId())
	}

	infos, missing, err := ds.clusterHealthStorage.GetMany(ctx, ids)
	if err != nil {
		log.Errorf("failed to populate health info for %d clusters: %v", len(ids), err)
		return
	}
	if len(infos) == 0 {
		return
	}

	missCount := 0
	healthIdx := 0
	for clusterIdx, cluster := range clusters {
		if missCount < len(missing) && clusterIdx == missing[missCount] {
			missCount++
			continue
		}
		cluster.HealthStatus = infos[healthIdx]
		healthIdx++
	}
}

func (ds *datastoreImpl) updateClusterNoLock(ctx context.Context, cluster *storage.Cluster) error {
	if err := normalizeCluster(cluster); err != nil {
		return err
	}
	if err := validateInput(cluster); err != nil {
		return err
	}

	if err := ds.clusterStorage.Upsert(ctx, cluster); err != nil {
		return err
	}
	ds.idToNameCache.Add(cluster.GetId(), cluster.GetName())
	ds.nameToIDCache.Add(cluster.GetName(), cluster.GetId())
	return nil
}

func (ds *datastoreImpl) LookupOrCreateClusterFromConfig(ctx context.Context, clusterID, bundleID string, hello *central.SensorHello) (*storage.Cluster, error) {
	if err := checkWriteSac(ctx, clusterID); err != nil {
		return nil, err
	}

	helmConfig := hello.GetHelmManagedConfigInit()
	manager := helmConfig.GetManagedBy()

	// Be backwards compatible for older Helm charts, which do not send the `managedBy` field: derive `managedBy` from the provided `notHelmManaged` property.
	if manager == storage.ManagerType_MANAGER_TYPE_UNKNOWN && helmConfig.GetNotHelmManaged() {
		manager = storage.ManagerType_MANAGER_TYPE_MANUAL
	}

	ds.lock.Lock()
	defer ds.lock.Unlock()

	clusterName := helmConfig.GetClusterName()

	if clusterID == "" && clusterName != "" {
		// Try to look up cluster ID by name, if this is for an existing cluster
		clusterIDVal, _ := ds.nameToIDCache.Get(clusterName)
		clusterID, _ = clusterIDVal.(string)
	}

	isExisting := false
	var cluster *storage.Cluster
	if clusterID != "" {
		clusterByID, exist, err := ds.GetCluster(ctx, clusterID)
		if err != nil {
			return nil, err
		} else if !exist {
			return nil, errors.Errorf("cluster with ID %q does not exist", clusterID)
		}

		isExisting = true

		// If a name is specified, validate it (otherwise, accept any name)
		if clusterName != "" && clusterName != clusterByID.GetName() {
			return nil, errors.Errorf("Name mismatch for cluster %q: expected %q, but %q was specified. Set the cluster.name/clusterName attribute in your Helm config to %q, or remove it", clusterID, cluster.GetName(), clusterName, cluster.GetName())
		}

		cluster = clusterByID
	} else if clusterName != "" {
		// A this point, we can be sure that the cluster does not exist.
		cluster = &storage.Cluster{
			Name:               clusterName,
			InitBundleId:       bundleID,
			MostRecentSensorId: hello.GetDeploymentIdentification().Clone(),
		}
		clusterConfig := helmConfig.GetClusterConfig()
		configureFromHelmConfig(cluster, clusterConfig)

		// Unless we know for sure that we are not Helm-managed we do store the Helm configuration,
		// in particular this also applies to the UNKNOWN case.
		if manager != storage.ManagerType_MANAGER_TYPE_MANUAL {
			cluster.HelmConfig = clusterConfig.Clone()
		}

		if _, err := ds.addClusterNoLock(ctx, cluster); err != nil {
			return nil, errors.Wrapf(err, "failed to dynamically add cluster with name %q", clusterName)
		}
	} else {
		return nil, errors.New("neither a cluster ID nor a cluster name was specified")
	}

	if manager != storage.ManagerType_MANAGER_TYPE_MANUAL && isExisting {
		// This is short-cut for clusters whose Helm config fingerprint and init bundle ID is unchanged.
		// Applies to Helm- and Operator-managed clusters, not to manually managed clusters.

		// Check if the newly incoming request may replace the old connection
		lastContact := protoconv.ConvertTimestampToTimeOrDefault(cluster.GetHealthStatus().GetLastContact(), time.Time{})
		timeLeftInGracePeriod := clusterMoveGracePeriod - time.Since(lastContact)

		// In a scale test environment, allow Sensors to reconnect in under the time limit
		if timeLeftInGracePeriod > 0 && !env.ScaleTestEnabled.BooleanSetting() {
			if err := common.CheckConnReplace(hello.GetDeploymentIdentification(), cluster.GetMostRecentSensorId()); err != nil {
				managerPretty := "non-manually" // Unless we extend the `ManagerType` and forget to extend the switch here, this should never surface to the user.
				switch manager {
				case storage.ManagerType_MANAGER_TYPE_HELM_CHART:
					managerPretty = "Helm"
				case storage.ManagerType_MANAGER_TYPE_KUBERNETES_OPERATOR:
					managerPretty = "Operator"
				}
				return nil, errors.Errorf("registering %s-managed cluster is not allowed: %s. If you recently re-deployed, please wait for another %v",
					managerPretty, err, timeLeftInGracePeriod)
			}
		}

		if cluster.GetInitBundleId() == bundleID &&
			cluster.GetHelmConfig().GetConfigFingerprint() == helmConfig.GetClusterConfig().GetConfigFingerprint() &&
			cluster.GetManagedBy() == manager {
			// No change in either of
			// * fingerprint of the Helm configuration
			// * in init bundle ID
			// * manager type
			//
			// => there is no need to update the cluster, return immediately.
			//
			// Note: this also is the case if the cluster was newly added.
			return cluster, nil
		}
	}

	clusterConfig := helmConfig.GetClusterConfig()
	currentCluster := cluster

	cluster = cluster.Clone()
	cluster.ManagedBy = manager
	cluster.InitBundleId = bundleID
	if manager == storage.ManagerType_MANAGER_TYPE_MANUAL {
		cluster.HelmConfig = nil
	} else {
		configureFromHelmConfig(cluster, clusterConfig)
		cluster.HelmConfig = clusterConfig.Clone()
	}

	if !proto.Equal(currentCluster, cluster) {
		// Cluster is dirty and needs to be updated in the DB.
		if err := ds.updateClusterNoLock(ctx, cluster); err != nil {
			return nil, err
		}
	}

	return cluster, nil
}

func normalizeCluster(cluster *storage.Cluster) error {
	if cluster == nil {
		return errox.InvariantViolation.CausedBy("cannot normalize nil cluster object")
	}

	cluster.CentralApiEndpoint = strings.TrimPrefix(cluster.GetCentralApiEndpoint(), "https://")
	cluster.CentralApiEndpoint = strings.TrimPrefix(cluster.GetCentralApiEndpoint(), "http://")

	return addDefaults(cluster)
}

func validateInput(cluster *storage.Cluster) error {
	return clusterValidation.Validate(cluster).ToError()
}

// addDefaults enriches the provided non-nil cluster object with defaults for
// fields that cannot stay empty.
// `cluster.* bool` flags remain untouched.
func addDefaults(cluster *storage.Cluster) error {
	if cluster == nil {
		return errox.InvariantViolation.CausedBy("cannot enrich nil cluster object")
	}

	collectionMethod := cluster.GetCollectionMethod()

	// For backwards compatibility reasons, if Collection Method is not set, or set
	// to KERNEL_MODULE (which is unsupported) then honor defaults for runtime support
	if collectionMethod == storage.CollectionMethod_UNSET_COLLECTION || collectionMethod == storage.CollectionMethod_KERNEL_MODULE {
		cluster.CollectionMethod = storage.CollectionMethod_EBPF
	}
	cluster.RuntimeSupport = cluster.GetCollectionMethod() != storage.CollectionMethod_NO_COLLECTION

	if cluster.GetTolerationsConfig() == nil {
		cluster.TolerationsConfig = &storage.TolerationsConfig{
			Disabled: false,
		}
	}

	if cluster.GetDynamicConfig() == nil {
		cluster.DynamicConfig = &storage.DynamicClusterConfig{}
	}
	if cluster.GetType() != storage.ClusterType_OPENSHIFT4_CLUSTER {
		cluster.DynamicConfig.DisableAuditLogs = true
	}

	acConfig := cluster.DynamicConfig.GetAdmissionControllerConfig()
	if acConfig == nil {
		acConfig = &storage.AdmissionControllerConfig{
			Enabled: false,
		}
		cluster.DynamicConfig.AdmissionControllerConfig = acConfig
	}
	if acConfig.GetTimeoutSeconds() < 0 {
		return fmt.Errorf("timeout of %d is invalid", acConfig.GetTimeoutSeconds())
	}
	if acConfig.GetTimeoutSeconds() == 0 {
		acConfig.TimeoutSeconds = defaultAdmissionControllerTimeout
	}
	if cluster.GetMainImage() == "" {
		flavor := defaults.GetImageFlavorFromEnv()
		cluster.MainImage = flavor.MainImageNoTag()
		// cluster.CollectorImage should be kept empty here on the save path
		// because it is computed using complex rules from the MainImage on the load path.
	}
	if cluster.GetCentralApiEndpoint() == "" {
		cluster.CentralApiEndpoint = "central.stackrox:443"
	}
	return nil
}

func configureFromHelmConfig(cluster *storage.Cluster, helmConfig *storage.CompleteClusterConfig) {
	cluster.DynamicConfig = helmConfig.GetDynamicConfig().Clone()

	staticConfig := helmConfig.GetStaticConfig()
	cluster.Labels = helmConfig.GetClusterLabels()
	cluster.Type = staticConfig.GetType()
	cluster.MainImage = staticConfig.GetMainImage()
	cluster.CentralApiEndpoint = staticConfig.GetCentralApiEndpoint()
	cluster.CollectionMethod = staticConfig.GetCollectionMethod()
	cluster.CollectorImage = staticConfig.GetCollectorImage()
	cluster.AdmissionController = staticConfig.GetAdmissionController()
	cluster.AdmissionControllerUpdates = staticConfig.GetAdmissionControllerUpdates()
	cluster.AdmissionControllerEvents = staticConfig.GetAdmissionControllerEvents()
	cluster.TolerationsConfig = staticConfig.GetTolerationsConfig().Clone()
	cluster.SlimCollector = staticConfig.GetSlimCollector()
}

func (ds *datastoreImpl) collectClusters(ctx context.Context) ([]*storage.Cluster, error) {
	var clusters []*storage.Cluster
	walkFn := func() error {
		clusters = clusters[:0]
		return ds.clusterStorage.Walk(ctx, func(cluster *storage.Cluster) error {
			clusters = append(clusters, cluster)
			return nil
		})
	}
	if err := pgutils.RetryIfPostgres(walkFn); err != nil {
		return nil, err
	}
	return clusters, nil
}
