package pruning

import (
	"context"
	"time"

	"github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
	alertDatastore "github.com/stackrox/rox/central/alert/datastore"
	clusterDatastore "github.com/stackrox/rox/central/cluster/datastore"
	configDatastore "github.com/stackrox/rox/central/config/datastore"
	deploymentDatastore "github.com/stackrox/rox/central/deployment/datastore"
	imageDatastore "github.com/stackrox/rox/central/image/datastore"
	imageComponentDatastore "github.com/stackrox/rox/central/imagecomponent/datastore"
	networkFlowDatastore "github.com/stackrox/rox/central/networkgraph/flow/datastore"
	nodeGlobalDatastore "github.com/stackrox/rox/central/node/globaldatastore"
	podDatastore "github.com/stackrox/rox/central/pod/datastore"
	processBaselineDatastore "github.com/stackrox/rox/central/processbaseline/datastore"
	processDatastore "github.com/stackrox/rox/central/processindicator/datastore"
	k8sRoleDataStore "github.com/stackrox/rox/central/rbac/k8srole/datastore"
	roleBindingDataStore "github.com/stackrox/rox/central/rbac/k8srolebinding/datastore"
	riskDataStore "github.com/stackrox/rox/central/risk/datastore"
	serviceAccountDataStore "github.com/stackrox/rox/central/serviceaccount/datastore"
	vulnReqDataStore "github.com/stackrox/rox/central/vulnerabilityrequest/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/protoutils"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sliceutils"
	"github.com/stackrox/rox/pkg/timeutil"
)

const (
	pruneInterval      = 1 * time.Hour
	orphanWindow       = 30 * time.Minute
	baselineBatchLimit = 10000
	clusterGCFreq      = 24 * time.Hour
)

var (
	log                  = logging.LoggerForModule()
	pruningCtx           = sac.WithAllAccess(context.Background())
	lastClusterPruneTime time.Time
)

// GarbageCollector implements a generic garbage collection mechanism
type GarbageCollector interface {
	Start()
	Stop()
}

func newGarbageCollector(alerts alertDatastore.DataStore,
	nodes nodeGlobalDatastore.GlobalDataStore,
	images imageDatastore.DataStore,
	clusters clusterDatastore.DataStore,
	deployments deploymentDatastore.DataStore,
	pods podDatastore.DataStore,
	processes processDatastore.DataStore,
	processbaseline processBaselineDatastore.DataStore,
	networkflows networkFlowDatastore.ClusterDataStore,
	config configDatastore.DataStore,
	imageComponents imageComponentDatastore.DataStore,
	risks riskDataStore.DataStore,
	vulnReqs vulnReqDataStore.DataStore,
	serviceAccts serviceAccountDataStore.DataStore,
	k8sRoles k8sRoleDataStore.DataStore,
	k8sRoleBindings roleBindingDataStore.DataStore) GarbageCollector {
	return &garbageCollectorImpl{
		alerts:          alerts,
		clusters:        clusters,
		nodes:           nodes,
		images:          images,
		imageComponents: imageComponents,
		deployments:     deployments,
		pods:            pods,
		processes:       processes,
		processbaseline: processbaseline,
		networkflows:    networkflows,
		config:          config,
		risks:           risks,
		vulnReqs:        vulnReqs,
		serviceAccts:    serviceAccts,
		k8sRoles:        k8sRoles,
		k8sRoleBindings: k8sRoleBindings,
		stopSig:         concurrency.NewSignal(),
		stoppedSig:      concurrency.NewSignal(),
	}
}

type garbageCollectorImpl struct {
	alerts          alertDatastore.DataStore
	clusters        clusterDatastore.DataStore
	nodes           nodeGlobalDatastore.GlobalDataStore
	images          imageDatastore.DataStore
	imageComponents imageComponentDatastore.DataStore
	deployments     deploymentDatastore.DataStore
	pods            podDatastore.DataStore
	processes       processDatastore.DataStore
	processbaseline processBaselineDatastore.DataStore
	networkflows    networkFlowDatastore.ClusterDataStore
	config          configDatastore.DataStore
	risks           riskDataStore.DataStore
	vulnReqs        vulnReqDataStore.DataStore
	serviceAccts    serviceAccountDataStore.DataStore
	k8sRoles        k8sRoleDataStore.DataStore
	k8sRoleBindings roleBindingDataStore.DataStore
	stopSig         concurrency.Signal
	stoppedSig      concurrency.Signal
}

func (g *garbageCollectorImpl) Start() {
	go g.runGC()
}

func (g *garbageCollectorImpl) pruneBasedOnConfig() {
	config, err := g.config.GetConfig(pruningCtx)
	if err != nil {
		log.Error(err)
		return
	}
	if config == nil {
		log.Error("UNEXPECTED: Got nil config")
		return
	}
	log.Info("[Pruning] Starting a garbage collection cycle")
	pvtConfig := config.GetPrivateConfig()
	g.collectImages(pvtConfig)
	g.collectAlerts(pvtConfig)
	g.removeOrphanedResources()
	g.removeOrphanedRisks()
	g.removeExpiredVulnRequests()
	if features.DecommissionedClusterRetention.Enabled() {
		g.collectClusters(pvtConfig)
	}

	log.Info("[Pruning] Finished garbage collection cycle")
}

func (g *garbageCollectorImpl) runGC() {
	lastClusterPruneTime = time.Now().Add(-24 * time.Hour)
	g.pruneBasedOnConfig()

	t := time.NewTicker(pruneInterval)
	for {
		select {
		case <-t.C:
			g.pruneBasedOnConfig()
		case <-g.stopSig.Done():
			g.stoppedSig.Signal()
			return
		}
	}
}

// Remove vulnerability requests that have expired and past the retention period.
func (g *garbageCollectorImpl) removeExpiredVulnRequests() {
	results, err := g.vulnReqs.Search(
		pruningCtx,
		search.ConjunctionQuery(
			search.NewQueryBuilder().AddBools(search.ExpiredRequest, true).ProtoQuery(),
			search.NewQueryBuilder().AddDays(search.LastUpdatedTime, int64(configDatastore.DefaultExpiredVulnReqRetention)).ProtoQuery()),
	)
	if err != nil {
		log.Errorf("Error fetching expired vulnerability requests for pruning: %v", err)
		return
	}
	if len(results) == 0 {
		return
	}

	log.Infof("[Pruning] Found %d expired vulnerability requests. Deleting...", len(results))

	if err := g.vulnReqs.RemoveRequestsInternal(pruningCtx, search.ResultsToIDs(results)); err != nil {
		log.Errorf("Failed to remove some expired vulnerability requests. Removal will be retried in next pruning cycle: %v", err)
	}
}

// Remove pods where the cluster has been deleted.
func (g *garbageCollectorImpl) removeOrphanedPods(clusters set.FrozenStringSet) {
	var podIDsToRemove []string
	staleClusterIDsFound := set.NewStringSet()
	err := g.pods.WalkAll(pruningCtx, func(pod *storage.Pod) error {
		if !clusters.Contains(pod.GetClusterId()) {
			podIDsToRemove = append(podIDsToRemove, pod.GetId())
			staleClusterIDsFound.Add(pod.GetClusterId())
		}
		return nil
	})
	if err != nil {
		log.Errorf("Error walking pods to find orphaned pods: %v", err)
		return
	}
	if len(podIDsToRemove) == 0 {
		log.Info("[Pruning] Found no orphaned pods...")
		return
	}
	log.Infof("[Pruning] Found %d orphaned pods (from formerly deleted clusters: %v). Deleting...",
		len(podIDsToRemove), staleClusterIDsFound.AsSlice())

	for _, id := range podIDsToRemove {
		if err := g.pods.RemovePod(pruningCtx, id); err != nil {
			log.Errorf("Failed to remove pod with id %s: %v", id, err)
		}
	}
}

func removeOrphanedObjectsBySearch(searchQuery *v1.Query, name string, searchFn func(ctx context.Context, query *v1.Query) ([]search.Result, error), removeFn func(ctx context.Context, id string) error) {
	searchRes, err := searchFn(pruningCtx, searchQuery)
	if err != nil {
		log.Errorf("Error finding orphaned %s: %v", name, err)
		return
	}
	if len(searchRes) == 0 {
		log.Infof("[Pruning] Found no orphaned %s...", name)
		return
	}

	log.Infof("[Pruning] Found %d orphaned %s. Deleting...", len(searchRes), name)

	for _, res := range searchRes {
		if err := removeFn(pruningCtx, res.ID); err != nil {
			log.Errorf("Failed to remove %s with id %s: %v", name, res.ID, err)
		}
	}
}

// Remove ServiceAccounts where the cluster has been deleted.
func (g *garbageCollectorImpl) removeOrphanedServiceAccounts(searchQuery *v1.Query) {
	removeOrphanedObjectsBySearch(searchQuery, "service accounts", g.serviceAccts.Search, g.serviceAccts.RemoveServiceAccount)
}

// Remove K8SRoles where the cluster has been deleted.
func (g *garbageCollectorImpl) removeOrphanedK8SRoles(searchQuery *v1.Query) {
	removeOrphanedObjectsBySearch(searchQuery, "K8S roles", g.k8sRoles.Search, g.k8sRoles.RemoveRole)
}

// Remove K8SRoleBinding where the cluster has been deleted.
func (g *garbageCollectorImpl) removeOrphanedK8SRoleBindings(searchQuery *v1.Query) {
	removeOrphanedObjectsBySearch(searchQuery, "K8S role bindings", g.k8sRoleBindings.Search, g.k8sRoleBindings.RemoveRoleBinding)
}

func (g *garbageCollectorImpl) removeOrphanedResources() {
	clusters, err := g.clusters.GetClusters(pruningCtx)
	if err != nil {
		log.Errorf("Failed to fetch clusters: %v", err)
		return
	}
	clusterIDs := make([]string, 0, len(clusters))
	for _, c := range clusters {
		clusterIDs = append(clusterIDs, c.GetId())
	}
	clusterIDSet := set.NewFrozenStringSet(clusterIDs...)

	deploymentIDs, err := g.deployments.GetDeploymentIDs(pruningCtx)
	if err != nil {
		log.Error(errors.Wrap(err, "unable to fetch deployment IDs in pruning"))
		return
	}
	deploymentSet := set.NewFrozenStringSet(deploymentIDs...)
	podIDs, err := g.pods.GetPodIDs(pruningCtx)
	if err != nil {
		log.Error(errors.Wrap(err, "unable to fetch pod IDs in pruning"))
		return
	}
	g.removeOrphanedProcesses(deploymentSet, set.NewFrozenStringSet(podIDs...))
	g.removeOrphanedProcessBaselines(deploymentSet)
	g.markOrphanedAlertsAsResolved(deploymentSet)
	g.removeOrphanedNetworkFlows(deploymentSet, clusterIDSet)

	// TODO: Convert this from ListSearch to using negation search similar to SAs, roles and role bindings below
	g.removeOrphanedPods(clusterIDSet)

	q := clusterIDsToNegationQuery(clusterIDSet)
	g.removeOrphanedServiceAccounts(q)
	g.removeOrphanedK8SRoles(q)
	g.removeOrphanedK8SRoleBindings(q)
}

func clusterIDsToNegationQuery(clusterIDSet set.FrozenStringSet) *v1.Query {
	// TODO: When searching can be done with SQL, this should be refactored to a simple `NOT IN...` query. This current one is inefficent
	// with a large number of clusters and because of the required conjunction query that is taking a hit being a regex query to do nothing
	// Bleve/booleanquery requires a conjunction so it can't be removed
	var mustNot *v1.DisjunctionQuery
	if clusterIDSet.Cardinality() > 1 {
		mustNot = search.DisjunctionQuery(search.NewQueryBuilder().AddExactMatches(search.ClusterID, clusterIDSet.AsSlice()...).ProtoQuery()).GetDisjunction()
	} else {
		// Manually generating a disjunction because search.DisjunctionQuery returns a v1.Query if there's only thing it's matching on
		// which then results in a nil disjunction inside boolean query. That means this search will match everything.
		mustNot = (&v1.Query{
			Query: &v1.Query_Disjunction{Disjunction: &v1.DisjunctionQuery{
				Queries: []*v1.Query{search.NewQueryBuilder().AddExactMatches(search.ClusterID, clusterIDSet.AsSlice()...).ProtoQuery()},
			}},
		}).GetDisjunction()
	}

	must := (&v1.Query{
		// Similar to disjunction, conjunction needs multiple queries, or it has to be manually created
		// Unlike disjunction, if there's only one query when the boolean query is used it will panic
		Query: &v1.Query_Conjunction{Conjunction: &v1.ConjunctionQuery{
			Queries: []*v1.Query{search.NewQueryBuilder().AddStrings(search.ClusterID, search.WildcardString).ProtoQuery()},
		}},
	}).GetConjunction()

	return search.NewBooleanQuery(must, mustNot)
}

func (g *garbageCollectorImpl) removeOrphanedProcesses(deploymentIDs, podIDs set.FrozenStringSet) {
	var processesToPrune []string
	now := types.TimestampNow()
	err := g.processes.WalkAll(pruningCtx, func(pi *storage.ProcessIndicator) error {
		if pi.GetPodUid() != "" && podIDs.Contains(pi.GetPodUid()) {
			return nil
		}
		if pi.GetPodUid() == "" && deploymentIDs.Contains(pi.GetDeploymentId()) {
			return nil
		}
		if protoutils.Sub(now, pi.GetSignal().GetTime()) < orphanWindow {
			return nil
		}
		processesToPrune = append(processesToPrune, pi.GetId())
		return nil
	})
	if err != nil {
		log.Error(errors.Wrap(err, "unable to walk processes and mark for pruning"))
		return
	}
	log.Infof("[Process pruning] Found %d orphaned processes", len(processesToPrune))
	if err := g.processes.RemoveProcessIndicators(pruningCtx, processesToPrune); err != nil {
		log.Error(errors.Wrap(err, "error removing process indicators"))
	}
}

func (g *garbageCollectorImpl) removeOrphanedProcessBaselines(deployments set.FrozenStringSet) {
	var baselineBatchOffset, prunedProcessBaselines int32
	for {
		allQuery := &v1.Query{
			Pagination: &v1.QueryPagination{
				Offset: baselineBatchOffset,
				Limit:  baselineBatchLimit,
			},
		}

		res, err := g.processbaseline.Search(pruningCtx, allQuery)
		if err != nil {
			log.Error(errors.Wrap(err, "error searching process baselines"))
			return
		}

		baselineBatchOffset += baselineBatchLimit
		var baselineKeysToPrune []*storage.ProcessBaselineKey
		for _, baseline := range res {
			baselineKey, err := processBaselineDatastore.IDToKey(baseline.ID)
			if err != nil {
				log.Error(errors.Wrapf(err, "Invalid id %s", baseline.ID))
				continue
			}

			if !deployments.Contains(baselineKey.GetDeploymentId()) {
				baselineKeysToPrune = append(baselineKeysToPrune, baselineKey)
			}
		}

		now := types.TimestampNow()
		for _, baselineKey := range baselineKeysToPrune {
			baseline, exists, err := g.processbaseline.GetProcessBaseline(pruningCtx, baselineKey)
			if err != nil {
				log.Error(errors.Wrapf(err, "unable to fetch process baseline for key %v", baselineKey))
				continue
			}

			if !exists || protoutils.Sub(now, baseline.GetCreated()) < orphanWindow {
				continue
			}

			if err = g.processbaseline.RemoveProcessBaseline(pruningCtx, baselineKey); err != nil {
				log.Error(errors.Wrapf(err, "unable to remove process baseline: %v", baselineKey))
				continue
			}

			prunedProcessBaselines++
		}

		if len(res) < baselineBatchLimit {
			break
		}
	}

	log.Infof("[Process baseline pruning] Removed %d process baselines", prunedProcessBaselines)
}

func (g *garbageCollectorImpl) markOrphanedAlertsAsResolved(deployments set.FrozenStringSet) {
	var alertsToResolve []string
	now := types.TimestampNow()
	err := g.alerts.WalkAll(pruningCtx, func(alert *storage.ListAlert) error {
		// We should only remove orphaned deploy time alerts as they are not cleaned up by retention policies
		// This will only happen when there is data inconsistency
		if alert.GetLifecycleStage() != storage.LifecycleStage_DEPLOY {
			return nil
		}
		if alert.GetState() != storage.ViolationState_ACTIVE {
			return nil
		}
		if deployments.Contains(alert.GetDeployment().GetId()) {
			return nil
		}
		if protoutils.Sub(now, alert.GetTime()) < orphanWindow {
			return nil
		}
		alertsToResolve = append(alertsToResolve, alert.GetId())
		return nil
	})
	if err != nil {
		log.Error(errors.Wrap(err, "unable to walk alerts and mark for pruning"))
		return
	}
	log.Infof("[Alert pruning] Found %d orphaned alerts", len(alertsToResolve))
	for _, a := range alertsToResolve {
		if err := g.alerts.MarkAlertStale(pruningCtx, a); err != nil {
			log.Error(errors.Wrapf(err, "error marking alert %q as stale", a))
		}
	}
}

func isOrphanedDeployment(deployments set.FrozenStringSet, info *storage.NetworkEntityInfo) bool {
	return info.GetType() == storage.NetworkEntityInfo_DEPLOYMENT && !deployments.Contains(info.GetId())
}

func (g *garbageCollectorImpl) removeOrphanedNetworkFlows(deployments, clusters set.FrozenStringSet) {
	for _, c := range clusters.AsSlice() {
		store, err := g.networkflows.GetFlowStore(pruningCtx, c)
		if err != nil {
			log.Errorf("error getting flow store for cluster %q: %v", c, err)
			continue
		} else if store == nil {
			continue
		}
		now := types.TimestampNow()

		keyMatchFn := func(props *storage.NetworkFlowProperties) bool {
			return isOrphanedDeployment(deployments, props.GetSrcEntity()) ||
				isOrphanedDeployment(deployments, props.GetDstEntity())
		}
		valueMatchFn := func(flow *storage.NetworkFlow) bool {
			return flow.LastSeenTimestamp != nil && protoutils.Sub(now, flow.LastSeenTimestamp) > orphanWindow
		}
		err = store.RemoveMatchingFlows(pruningCtx, keyMatchFn, valueMatchFn)
		if err != nil {
			log.Errorf("error removing orphaned flows for cluster %q: %v", c, err)
		}
	}
}

func (g *garbageCollectorImpl) collectImages(config *storage.PrivateConfig) {
	pruneImageAfterDays := config.GetImageRetentionDurationDays()
	qb := search.NewQueryBuilder().AddDays(search.LastUpdatedTime, int64(pruneImageAfterDays)).ProtoQuery()
	imageResults, err := g.images.Search(pruningCtx, qb)
	if err != nil {
		log.Error(err)
		return
	}
	log.Infof("[Image pruning] Found %d image search results", len(imageResults))

	imagesToPrune := make([]string, 0, len(imageResults))
	for _, result := range imageResults {
		q1 := search.NewQueryBuilder().AddExactMatches(search.ImageSHA, result.ID).ProtoQuery()
		q2 := search.NewQueryBuilder().AddExactMatches(search.ContainerImageDigest, result.ID).ProtoQuery()
		deploymentResults, err := g.deployments.Search(pruningCtx, q1)
		if err != nil {
			log.Errorf("[Image pruning] searching deployments: %v", err)
			continue
		}

		podResults, err := g.pods.Search(pruningCtx, q2)
		if err != nil {
			log.Errorf("[Image pruning] searching pods: %v", err)
			continue
		}

		if len(deploymentResults) == 0 && len(podResults) == 0 {
			imagesToPrune = append(imagesToPrune, result.ID)
		}
	}
	if len(imagesToPrune) > 0 {
		log.Infof("[Image Pruning] Removing %d images", len(imagesToPrune))
		log.Debugf("[Image Pruning] Removing images %+v", imagesToPrune)
		if err := g.images.DeleteImages(pruningCtx, imagesToPrune...); err != nil {
			log.Error(err)
		}
	}
}

func (g *garbageCollectorImpl) collectClusters(config *storage.PrivateConfig) {
	if !features.DecommissionedClusterRetention.Enabled() {
		return
	}

	// Check to see if pruning is enabled
	clusterRetention := config.GetDecommissionedClusterRetention()
	retentionDays := int64(clusterRetention.GetRetentionDurationDays())
	if retentionDays == 0 {
		log.Info("[Cluster Pruning] pruning is disabled.")
		return
	}

	// Check to see if enough time has elapsed to run again
	if lastClusterPruneTime.Add(clusterGCFreq).After(time.Now()) {
		// Only cluster pruning if it's been at least clusterGCFreq since last run
		return
	}
	defer func() {
		lastClusterPruneTime = time.Now()
	}()

	// Allow 24hrs grace period after the config changes
	lastUpdateTime, err := types.TimestampFromProto(clusterRetention.GetLastUpdated())
	if err != nil {
		log.Error(err)
		return
	}
	if timeutil.TimeDiffDays(time.Now(), lastUpdateTime) < 1 {
		// Allow 24 hr grace period after a change in config
		log.Info("[Cluster Pruning] skipping pruning to allow 24 hr grace period after a recent change in cluster retention config.")
		return
	}

	// Retention should start counting _after_ the config is created (which is basically when upgraded to 71)
	configCreationTime, err := types.TimestampFromProto(clusterRetention.GetCreatedAt())
	if err != nil {
		log.Error(err)
		return
	}
	if timeutil.TimeDiffDays(time.Now(), configCreationTime) < int(retentionDays) {
		// In this case, the deployments that became unhealthy after config creation would also be unhealthy for fewer than retention days
		// and pruning can be skipped
		log.Info("[Cluster Pruning] skipping pruning as retention period only starts after upgrade to version 3.71.0.")
		return
	}

	allClusterCount, err := g.clusters.CountClusters(pruningCtx)
	if err != nil {
		log.Errorf("[Cluster Pruning] error counting clusters: %v", err)
	}

	if allClusterCount == 0 {
		log.Info("[Cluster Pruning] skipping pruning because no clusters were found.")
		return
	}

	query := search.NewQueryBuilder().
		AddDays(search.LastContactTime, retentionDays).
		AddExactMatches(search.SensorStatus, storage.ClusterHealthStatus_UNHEALTHY.String()).ProtoQuery()
	clusters, err := g.clusters.SearchRawClusters(pruningCtx, query)
	if err != nil {
		log.Errorf("[Cluster Pruning] error searching for clusters: %v", err)
		return
	}

	log.Infof("[Cluster Pruning] found %d cluster(s) that haven't been active in over %d days", len(clusters), retentionDays)

	clustersToPrune := make([]string, 0)
	for _, cluster := range clusters {
		if sliceutils.MapsIntersect(clusterRetention.GetIgnoreClusterLabels(), cluster.GetLabels()) {
			log.Infof("[Cluster Pruning] skipping excluded cluster with id %s", cluster.GetId())
			continue
		}

		// Don't delete if the cluster contains central
		hasCentral, err := g.checkIfClusterContainsCentral(cluster)
		if err != nil {
			log.Errorf("[Cluster Pruning] error searching for deployements in cluster: %v", err)
			return
		}
		if hasCentral {
			// Warning because it's important to know that your central cluster is unhealthy
			log.Warnf("[Cluster Pruning] skipping pruning cluster with id %s because this cluster contains the central deployment.", cluster.GetId())
			continue
		}
		clustersToPrune = append(clustersToPrune, cluster.GetId())
	}

	if allClusterCount == len(clustersToPrune) {
		log.Warnf("[Cluster Pruning] skipping pruning because all %d cluster(s) are unhealthy and this is an abnormal state. Please remove any manually if desired.", allClusterCount)
		return
	}

	if len(clustersToPrune) == 0 {
		// Debug log as it's be noisy, but is helpful if debugging
		log.Debug("[Cluster Pruning] no inactive clusters found.")
		return
	}

	for _, clusterId := range clustersToPrune {
		log.Infof("[Cluster Pruning] Removing cluster with ID %s", clusterId)
		if err := g.clusters.RemoveCluster(pruningCtx, clusterId, nil); err != nil {
			log.Error(err)
			return
		}
	}
}

func (g *garbageCollectorImpl) checkIfClusterContainsCentral(cluster *storage.Cluster) (bool, error) {
	//This query could be expensive, but it's a rare occurrence. It only happens if there is a cluster that has been unhealthy for a long time (configurable)
	query := search.NewQueryBuilder().
		AddStrings(search.ClusterID, cluster.GetId()).
		AddExactMatches(search.DeploymentName, "central")
	deploys, err := g.deployments.SearchRawDeployments(pruningCtx, query.ProtoQuery())
	if err != nil {
		return false, err
	}

	// As long as we find at least one deployment that matches the criteria
	// This is meant to be tolerant of false positives while extremely intolerant of false negatives. Rather not delete a cluster than to delete one with central in it.
	for _, d := range deploys {
		// While this could be done via the searcher, this was moved out to reduce complexity of the search.
		// The search might run more often, but this manual grep only happens if the cluster is unhealthy AND has >= 1 deployment with name central.
		if d.GetLabels()["app"] == "central" && d.GetAnnotations()["owner"] == "stackrox" {
			return true, nil
		}
	}

	// TODO: We need to mark a cluster as having central going forward. Perhaps the above heuristic can be used to tag from within sensor.

	return false, nil
}

func getConfigValues(config *storage.PrivateConfig) (pruneResolvedDeployAfter, pruneAllRuntimeAfter, pruneDeletedRuntimeAfter, pruneAttemptedDeployAfter, pruneAttemptedRuntimeAfter int32) {
	alertRetention := config.GetAlertRetention()
	if val, ok := alertRetention.(*storage.PrivateConfig_DEPRECATEDAlertRetentionDurationDays); ok {
		global := val.DEPRECATEDAlertRetentionDurationDays
		return global, global, global, global, global

	} else if val, ok := alertRetention.(*storage.PrivateConfig_AlertConfig); ok {
		return val.AlertConfig.GetResolvedDeployRetentionDurationDays(),
			val.AlertConfig.GetAllRuntimeRetentionDurationDays(),
			val.AlertConfig.GetDeletedRuntimeRetentionDurationDays(),
			val.AlertConfig.GetAttemptedDeployRetentionDurationDays(),
			val.AlertConfig.GetAttemptedRuntimeRetentionDurationDays()
	}
	return 0, 0, 0, 0, 0
}

func (g *garbageCollectorImpl) collectAlerts(config *storage.PrivateConfig) {
	alertRetention := config.GetAlertRetention()
	if alertRetention == nil {
		log.Info("[Alert pruning] Alert pruning has been disabled.")
		return
	}

	pruneResolvedDeployAfter,
		pruneAllRuntimeAfter,
		pruneDeletedRuntimeAfter,
		pruneAttemptedDeployAfter,
		pruneAttemptedRuntimeAfter := getConfigValues(config)

	var queries []*v1.Query

	if pruneResolvedDeployAfter > 0 {
		q := search.NewQueryBuilder().
			AddStrings(search.LifecycleStage, storage.LifecycleStage_DEPLOY.String()).
			AddStrings(search.ViolationState, storage.ViolationState_RESOLVED.String()).
			AddDays(search.ViolationTime, int64(pruneResolvedDeployAfter)).
			ProtoQuery()
		queries = append(queries, q)
	}

	if pruneAllRuntimeAfter > 0 {
		q := search.NewQueryBuilder().
			AddStrings(search.LifecycleStage, storage.LifecycleStage_RUNTIME.String()).
			AddDays(search.ViolationTime, int64(pruneAllRuntimeAfter)).
			ProtoQuery()
		queries = append(queries, q)
	}

	if pruneDeletedRuntimeAfter > 0 && pruneAllRuntimeAfter != pruneDeletedRuntimeAfter {
		q := search.NewQueryBuilder().
			AddStrings(search.LifecycleStage, storage.LifecycleStage_RUNTIME.String()).
			AddDays(search.ViolationTime, int64(pruneDeletedRuntimeAfter)).
			AddBools(search.Inactive, true).
			ProtoQuery()
		queries = append(queries, q)
	}

	if pruneAttemptedDeployAfter > 0 {
		q := search.NewQueryBuilder().
			AddStrings(search.LifecycleStage, storage.LifecycleStage_DEPLOY.String()).
			AddStrings(search.ViolationState, storage.ViolationState_ATTEMPTED.String()).
			AddDays(search.ViolationTime, int64(pruneAttemptedDeployAfter)).
			ProtoQuery()
		queries = append(queries, q)
	}

	if pruneAttemptedRuntimeAfter > 0 {
		q := search.NewQueryBuilder().
			AddStrings(search.LifecycleStage, storage.LifecycleStage_RUNTIME.String()).
			AddStrings(search.ViolationState, storage.ViolationState_ATTEMPTED.String()).
			AddDays(search.ViolationTime, int64(pruneAttemptedRuntimeAfter)).
			ProtoQuery()
		queries = append(queries, q)
	}

	if len(queries) == 0 {
		log.Info("No alert retention configuration, skipping")
		return
	}

	alertResults, err := g.alerts.Search(pruningCtx, search.DisjunctionQuery(queries...))
	if err != nil {
		log.Error(err)
		return
	}

	alertsToPrune := search.ResultsToIDs(alertResults)

	if len(alertsToPrune) > 0 {
		log.Infof("[Alert pruning] Removing %d alerts", len(alertsToPrune))
		if err := g.alerts.DeleteAlerts(pruningCtx, alertsToPrune...); err != nil {
			log.Error(err)
		}
	}
}

func (g *garbageCollectorImpl) removeOrphanedRisks() {
	g.removeOrphanedDeploymentRisks()
	g.removeOrphanedImageRisks()
	g.removeOrphanedImageComponentRisks()
	g.removeOrphanedNodeRisks()
}

func (g *garbageCollectorImpl) removeOrphanedDeploymentRisks() {
	deploymentsWithRisk := g.getRisks(storage.RiskSubjectType_DEPLOYMENT)
	results, err := g.deployments.Search(pruningCtx, search.EmptyQuery())
	if err != nil {
		log.Errorf("[Risk pruning] Searching deployments: %v", err)
		return
	}

	prunable := deploymentsWithRisk.Difference(search.ResultsToIDSet(results)).AsSlice()
	log.Infof("[Risk pruning] Removing %d deployment risks", len(prunable))
	g.removeRisks(storage.RiskSubjectType_DEPLOYMENT, prunable...)
}

func (g *garbageCollectorImpl) removeOrphanedImageRisks() {
	imagesWithRisk := g.getRisks(storage.RiskSubjectType_IMAGE)
	results, err := g.images.Search(pruningCtx, search.EmptyQuery())
	if err != nil {
		log.Errorf("[Risk pruning] Searching images: %v", err)
		return
	}

	prunable := imagesWithRisk.Difference(search.ResultsToIDSet(results)).AsSlice()
	log.Infof("[Risk pruning] Removing %d image risks", len(prunable))
	g.removeRisks(storage.RiskSubjectType_IMAGE, prunable...)
}

func (g *garbageCollectorImpl) removeOrphanedImageComponentRisks() {
	componentsWithRisk := g.getRisks(storage.RiskSubjectType_IMAGE_COMPONENT)
	results, err := g.imageComponents.Search(pruningCtx, search.EmptyQuery())
	if err != nil {
		log.Errorf("[Risk pruning] Searching image components: %v", err)
		return
	}

	prunable := componentsWithRisk.Difference(search.ResultsToIDSet(results)).AsSlice()
	log.Infof("[Risk pruning] Removing %d image component risks", len(prunable))
	g.removeRisks(storage.RiskSubjectType_IMAGE_COMPONENT, prunable...)
}

func (g *garbageCollectorImpl) removeOrphanedNodeRisks() {
	nodesWithRisk := g.getRisks(storage.RiskSubjectType_NODE)
	results, err := g.nodes.Search(pruningCtx, search.EmptyQuery())
	if err != nil {
		log.Errorf("[Risk pruning] Searching nodes: %v", err)
		return
	}

	prunable := nodesWithRisk.Difference(search.ResultsToIDSet(results)).AsSlice()
	log.Infof("[Risk pruning] Removing %d node risks", len(prunable))
	g.removeRisks(storage.RiskSubjectType_NODE, prunable...)
}

func (g *garbageCollectorImpl) getRisks(riskType storage.RiskSubjectType) set.StringSet {
	risks := set.NewStringSet()
	results, err := g.risks.Search(pruningCtx, search.NewQueryBuilder().AddExactMatches(search.RiskSubjectType, riskType.String()).ProtoQuery())
	if err != nil {
		log.Error(err)
		return risks
	}

	for _, result := range results {
		_, id, err := riskDataStore.GetIDParts(result.ID)
		if err != nil {
			log.Error(err)
			continue
		}
		risks.Add(id)
	}
	return risks
}

func (g *garbageCollectorImpl) removeRisks(riskType storage.RiskSubjectType, ids ...string) {
	for _, id := range ids {
		if err := g.risks.RemoveRisk(pruningCtx, id, riskType); err != nil {
			log.Error(err)
		}
	}
}

func (g *garbageCollectorImpl) Stop() {
	g.stopSig.Signal()
	<-g.stoppedSig.Done()
}
