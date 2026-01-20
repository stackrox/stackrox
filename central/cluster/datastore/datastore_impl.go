package datastore

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/pkg/errors"
	alertDataStore "github.com/stackrox/rox/central/alert/datastore"
	clusterStore "github.com/stackrox/rox/central/cluster/store/cluster"
	clusterHealthStore "github.com/stackrox/rox/central/cluster/store/clusterhealth"
	compliancePruning "github.com/stackrox/rox/central/complianceoperator/v2/pruner"
	"github.com/stackrox/rox/central/convert/storagetoeffectiveaccessscope"
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
	"github.com/stackrox/rox/pkg/centralsensor"
	clusterValidation "github.com/stackrox/rox/pkg/cluster"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/images/defaults"
	notifierProcessor "github.com/stackrox/rox/pkg/notifier"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/effectiveaccessscope"
	"github.com/stackrox/rox/pkg/sac/resources"
	pkgSearch "github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/paginated"
	"github.com/stackrox/rox/pkg/search/sorted"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/simplecache"
	"github.com/stackrox/rox/pkg/sliceutils"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/uuid"
)

const (
	// clusterMoveGracePeriod determines the amount of time that has to pass before a (logical) StackRox cluster can
	// be moved to a different (physical) Kubernetes cluster.
	clusterMoveGracePeriod = 3 * time.Minute
)

const (
	defaultAdmissionControllerTimeout = 10
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
	compliancePruner          compliancePruning.Pruner
	cm                        connection.Manager
	networkBaselineMgr        networkBaselineManager.Manager

	notifier      notifierProcessor.Processor
	clusterRanker *ranking.Ranker

	idToNameCache simplecache.Cache
	nameToIDCache simplecache.Cache

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
			clusterHealthStatuses[healthInfo.GetId()] = healthInfo
			return nil
		})
	}
	if err := pgutils.RetryIfPostgres(ctx, walkFn); err != nil {
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
	// Need to check if we are sorting by priority.
	validPriorityQuery, err := sorted.IsValidPriorityQuery(q, pkgSearch.ClusterPriority)
	if err != nil {
		return nil, err
	}
	if validPriorityQuery {
		priorityQuery, reversed, err := sorted.RemovePrioritySortFromQuery(q, pkgSearch.ClusterPriority)
		if err != nil {
			return nil, err
		}
		results, err := ds.clusterStorage.Search(ctx, priorityQuery)
		if err != nil {
			return nil, err
		}

		sortedResults := sorted.SortResults(results, reversed, ds.clusterRanker)
		return paginated.PageResults(sortedResults, q)
	}

	return ds.clusterStorage.Search(ctx, q)
}

// Count returns the number of search results from the query
func (ds *datastoreImpl) Count(ctx context.Context, q *v1.Query) (int, error) {
	return ds.clusterStorage.Count(ctx, q)
}

func (ds *datastoreImpl) SearchResults(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error) {
	if q == nil {
		q = pkgSearch.EmptyQuery()
	}
	clonedQuery := q.CloneVT()
	selectSelects := []*v1.QuerySelect{
		pkgSearch.NewQuerySelect(pkgSearch.Cluster).Proto(),
	}
	clonedQuery.Selects = append(clonedQuery.GetSelects(), selectSelects...)
	results, err := ds.Search(ctx, clonedQuery)
	if err != nil {
		return nil, err
	}
	for i := range results {
		if results[i].FieldValues != nil {
			if nameVal, ok := results[i].FieldValues[strings.ToLower(pkgSearch.Cluster.String())]; ok {
				results[i].Name = nameVal
			}
		}
	}
	return pkgSearch.ResultsToSearchResultProtos(results, &ClusterSearchResultConverter{}), nil
}

func (ds *datastoreImpl) searchRawClusters(ctx context.Context, q *v1.Query) ([]*storage.Cluster, error) {
	var clusters []*storage.Cluster
	validPriorityQuery, err := sorted.IsValidPriorityQuery(q, pkgSearch.ClusterPriority)
	if err != nil {
		return nil, err
	}
	if validPriorityQuery {
		results, err := ds.Search(ctx, q)
		if err != nil {
			return nil, err
		}

		clusters, _, err = ds.clusterStorage.GetMany(ctx, pkgSearch.ResultsToIDs(results))
		if err != nil {
			return nil, err
		}
	} else {
		err = ds.clusterStorage.WalkByQuery(ctx, q, func(cluster *storage.Cluster) error {
			clusters = append(clusters, cluster)
			return nil
		})
		slices.SortFunc(clusters, func(a, b *storage.Cluster) int {
			return strings.Compare(a.GetName(), b.GetName())
		})
		if err != nil {
			return nil, err
		}
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

func (ds *datastoreImpl) GetClustersForSAC() ([]effectiveaccessscope.Cluster, error) {
	return storagetoeffectiveaccessscope.Clusters(ds.clusterStorage.GetAllFromCacheForSAC()), nil
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

func (ds *datastoreImpl) WalkClusters(ctx context.Context, fn func(obj *storage.Cluster) error) error {
	walkFn := func() error {
		return ds.clusterStorage.Walk(ctx, func(cluster *storage.Cluster) error {
			clonedCluster := cluster.CloneVT()
			ds.populateHealthInfos(ctx, clonedCluster)
			ds.updateClusterPriority(clonedCluster)
			return fn(clonedCluster)
		})
	}
	if err := pgutils.RetryIfPostgres(ctx, walkFn); err != nil {
		return err
	}
	return nil
}

func (ds *datastoreImpl) SearchRawClusters(ctx context.Context, q *v1.Query) ([]*storage.Cluster, error) {
	clusters, err := ds.searchRawClusters(ctx, q)
	if err != nil {
		return nil, err
	}
	return clusters, nil
}

func (ds *datastoreImpl) CountClusters(ctx context.Context) (int, error) {
	if _, err := clusterSAC.ReadAllowed(ctx); err != nil {
		return 0, err
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
	_, found := ds.nameToIDCache.Get(cluster.GetName())
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

	trackClusterRegistered(cluster)

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

	// Fetch the cluster and confirm it exists.
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

	// Remove the cluster health since the cluster no longer exists
	if err := ds.clusterHealthStorage.Delete(ctx, cluster.GetId()); err != nil {
		log.Errorf("failed to remove health status for cluster %s: %v", cluster.GetId(), err)
	}

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

	if err := ds.netFlowsDataStore.RemoveFlowStore(ctx, cluster.GetId()); err != nil {
		log.Errorf("failed to delete network flows for removed cluster %s: %v", cluster.GetId(), err)
	}

	if features.ComplianceEnhancements.Enabled() {
		ds.compliancePruner.RemoveComplianceResourcesByCluster(ctx, cluster.GetId())
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
	return ds.alertDataStore.SearchRawAlerts(ctx, q, true)
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
		if ds.notifier != nil {
			ds.notifier.ProcessAlert(ctx, resolvedAlert)
		}
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

// clusterConfigData holds extracted configuration from SensorHello.
// This allows pure functions to work with structured data instead of raw protobuf messages.
type clusterConfigData struct {
	helmManagedConfigInit    *central.HelmManagedConfigInit
	deploymentIdentification *storage.SensorDeploymentIdentification
	capabilities             []string
}

func (c clusterConfigData) clusterName() string {
	return c.helmManagedConfigInit.GetClusterName()
}

func (c clusterConfigData) manager() storage.ManagerType {
	return c.helmManagedConfigInit.GetManagedBy()
}

func (c clusterConfigData) helmConfig() *storage.CompleteClusterConfig {
	return c.helmManagedConfigInit.GetClusterConfig()
}

// extractClusterConfig extracts relevant configuration data from SensorHello.
// This is a pure function that performs the extraction once.
func extractClusterConfig(hello *central.SensorHello) clusterConfigData {
	return clusterConfigData{
		helmManagedConfigInit:    hello.GetHelmManagedConfigInit(),
		deploymentIdentification: hello.GetDeploymentIdentification(),
		capabilities:             hello.GetCapabilities(),
	}
}

// shouldUpdateCluster determines if an existing cluster needs updating.
// Returns true if any of: sensor capabilities, config fingerprint, init bundle ID, or manager type has changed.
func shouldUpdateCluster(existing *storage.Cluster, config clusterConfigData, registrantID string) bool {
	if !sensorCapabilitiesEqual(existing, config.capabilities) {
		return true
	}
	if existing.GetInitBundleId() != registrantID {
		return true
	}
	if existing.GetHelmConfig().GetConfigFingerprint() != config.helmConfig().GetConfigFingerprint() {
		return true
	}
	if existing.GetManagedBy() != config.manager() {
		return true
	}
	return false
}

// sensorCapabilitiesEqual is a helper that compares capabilities from cluster and config.
func sensorCapabilitiesEqual(cluster *storage.Cluster, capabilities []string) bool {
	return set.NewSet(cluster.GetSensorCapabilities()...).Equal(set.NewSet(capabilities...))
}

// validateClusterConfig validates that the cluster name matches expectations.
// For existing clusters, ensures name consistency if a name is specified.
func validateClusterConfig(clusterID, clusterName string, existing *storage.Cluster) error {
	if clusterName != "" && clusterName != existing.GetName() {
		return errors.Errorf("Name mismatch for cluster %q: expected %q, but %q was specified. Set the cluster.name/clusterName attribute in your Helm config to %q, or remove it",
			clusterID, existing.GetName(), clusterName, existing.GetName())
	}
	return nil
}

// buildClusterFromConfig builds a new cluster object from configuration.
// This does not persist the cluster, just constructs the object.
func buildClusterFromConfig(clusterName, registrantID string, config clusterConfigData) *storage.Cluster {
	cluster := &storage.Cluster{
		Name:               clusterName,
		InitBundleId:       registrantID,
		MostRecentSensorId: config.deploymentIdentification.CloneVT(),
		SensorCapabilities: sliceutils.CopySliceSorted(config.capabilities),
	}
	configureFromHelmConfig(cluster, config.helmConfig())

	if centralsensor.SecuredClusterIsNotManagedManually(config.helmManagedConfigInit) {
		cluster.HelmConfig = config.helmConfig().CloneVT()
	}

	return cluster
}

// applyConfigToCluster applies configuration updates to a cluster.
// Returns a new cluster object with updates applied (immutable pattern).
func applyConfigToCluster(cluster *storage.Cluster, config clusterConfigData, registrantID string) *storage.Cluster {
	updated := cluster.CloneVT()
	updated.ManagedBy = config.manager()
	updated.InitBundleId = registrantID
	updated.SensorCapabilities = sliceutils.CopySliceSorted(config.capabilities)

	if centralsensor.SecuredClusterIsNotManagedManually(config.helmManagedConfigInit) {
		configureFromHelmConfig(updated, config.helmConfig())
		updated.HelmConfig = config.helmConfig().CloneVT()
	} else {
		updated.HelmConfig = nil
	}

	return updated
}

// checkGracePeriodForReconnect checks if reconnection is allowed based on grace period.
// For Helm/Operator managed clusters, prevents cluster moves within the grace period.
func checkGracePeriodForReconnect(cluster *storage.Cluster, deploymentID *storage.SensorDeploymentIdentification, manager storage.ManagerType) error {
	lastContact := protoconv.ConvertTimestampToTimeOrDefault(cluster.GetHealthStatus().GetLastContact(), time.Time{})
	timeLeftInGracePeriod := clusterMoveGracePeriod - time.Since(lastContact)

	// In a scale test environment, allow Sensors to reconnect in under the time limit
	if timeLeftInGracePeriod > 0 && !env.ScaleTestEnabled.BooleanSetting() {
		if err := common.CheckConnReplace(deploymentID, cluster.GetMostRecentSensorId()); err != nil {
			managerPretty := "non-manually" // Unless we extend the `ManagerType` and forget to extend the switch here, this should never surface to the user.
			switch manager {
			case storage.ManagerType_MANAGER_TYPE_HELM_CHART:
				managerPretty = "Helm"
			case storage.ManagerType_MANAGER_TYPE_KUBERNETES_OPERATOR:
				managerPretty = "Operator"
			}
			return errors.Errorf("registering %s-managed cluster is not allowed: %s. If you recently re-deployed, please wait for another %v",
				managerPretty, err, timeLeftInGracePeriod)
		}
	}
	return nil
}

// lookupOrCreateCluster handles the lookup-or-create logic.
// Returns the cluster, a bool indicating whether it was an existing cluster (true) or newly created (false), and an error.
// The bool is important because existing clusters need grace period checks and update checks, while new clusters skip those.
func (ds *datastoreImpl) lookupOrCreateCluster(ctx context.Context, clusterID, clusterName, registrantID string, config clusterConfigData) (*storage.Cluster, bool, error) {
	// Try to resolve cluster ID from name if not provided
	if clusterID == "" && clusterName != "" {
		if cachedID, ok := ds.nameToIDCache.Get(clusterName); ok {
			clusterID = cachedID.(string)
		}
	}

	// Path 1: Lookup existing cluster by ID
	if clusterID != "" {
		cluster, exists, err := ds.GetCluster(ctx, clusterID)
		if err != nil {
			return nil, false, err
		}
		if !exists {
			return nil, false, errors.Errorf("cluster with ID %q does not exist", clusterID)
		}

		// Validate name match
		if err := validateClusterConfig(clusterID, clusterName, cluster); err != nil {
			return nil, false, err
		}

		return cluster, true, nil
	}

	// Path 2: Create new cluster by name
	if clusterName != "" {
		cluster := buildClusterFromConfig(clusterName, registrantID, config)

		if _, err := ds.addClusterNoLock(ctx, cluster); err != nil {
			return nil, false, errors.Wrapf(err, "failed to dynamically add cluster with name %q", clusterName)
		}

		return cluster, false, nil
	}

	// Path 3: Neither ID nor name provided
	return nil, false, errors.New("neither a cluster ID nor a cluster name was specified")
}

// registrantID can be the ID of an init bundle or of a CRS.
func (ds *datastoreImpl) LookupOrCreateClusterFromConfig(ctx context.Context, clusterID, registrantID string, hello *central.SensorHello) (*storage.Cluster, error) {
	if err := checkWriteSac(ctx, clusterID); err != nil {
		return nil, err
	}

	// Extract configuration (pure function)
	config := extractClusterConfig(hello)

	ds.lock.Lock()
	defer ds.lock.Unlock()

	// Lookup or create cluster
	cluster, isExisting, err := ds.lookupOrCreateCluster(ctx, clusterID, config.clusterName(), registrantID, config)
	if err != nil {
		return nil, err
	}

	// For existing clusters, check if update is needed
	if isExisting && config.manager() != storage.ManagerType_MANAGER_TYPE_MANUAL {
		// Check grace period
		if err := checkGracePeriodForReconnect(cluster, config.deploymentIdentification, config.manager()); err != nil {
			return nil, err
		}

		// Check if update needed
		if !shouldUpdateCluster(cluster, config, registrantID) {
			return cluster, nil
		}
	}

	// Apply configuration updates
	updatedCluster := applyConfigToCluster(cluster, config, registrantID)

	// Persist if changed
	if !cluster.EqualVT(updatedCluster) {
		if err := ds.updateClusterNoLock(ctx, updatedCluster); err != nil {
			return nil, err
		}
	}

	return updatedCluster, nil
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
		cluster.CollectionMethod = storage.CollectionMethod_CORE_BPF
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

	acConfig := cluster.GetDynamicConfig().GetAdmissionControllerConfig()
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
	cluster.DynamicConfig = helmConfig.GetDynamicConfig().CloneVT()

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
	cluster.TolerationsConfig = staticConfig.GetTolerationsConfig().CloneVT()
	cluster.SlimCollector = staticConfig.GetSlimCollector()
	cluster.AdmissionControllerFailOnError = false
	if features.AdmissionControllerConfig.Enabled() {
		cluster.AdmissionControllerFailOnError = staticConfig.GetAdmissionControllerFailOnError()
	}
}

func (ds *datastoreImpl) collectClusters(ctx context.Context) ([]*storage.Cluster, error) {
	var clusters []*storage.Cluster
	walkFn := func() error {
		clusters = clusters[:0]
		return ds.clusterStorage.Walk(ctx, func(cluster *storage.Cluster) error {
			clusters = append(clusters, cluster.CloneVT())
			return nil
		})
	}
	if err := pgutils.RetryIfPostgres(ctx, walkFn); err != nil {
		return nil, err
	}
	return clusters, nil
}

// ClusterSearchResultConverter implements search.SearchResultConverter for cluster search results.
// This enables single-pass query construction for SearchResult protos.
type ClusterSearchResultConverter struct{}

func (c *ClusterSearchResultConverter) BuildName(result *pkgSearch.Result) string {
	return result.Name
}

func (c *ClusterSearchResultConverter) BuildLocation(result *pkgSearch.Result) string {
	return fmt.Sprintf("/%s", result.Name)
}

func (c *ClusterSearchResultConverter) GetCategory() v1.SearchCategory {
	return v1.SearchCategory_CLUSTERS
}
