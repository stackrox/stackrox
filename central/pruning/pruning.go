package pruning

import (
	"context"
	"time"

	"github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
	alertDatastore "github.com/stackrox/rox/central/alert/datastore"
	blobDatastore "github.com/stackrox/rox/central/blob/datastore"
	clusterDatastore "github.com/stackrox/rox/central/cluster/datastore"
	configDatastore "github.com/stackrox/rox/central/config/datastore"
	deploymentDatastore "github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/globaldb"
	imageDatastore "github.com/stackrox/rox/central/image/datastore"
	imageComponentDatastore "github.com/stackrox/rox/central/imagecomponent/datastore"
	logimbueDataStore "github.com/stackrox/rox/central/logimbue/store"
	networkFlowDatastore "github.com/stackrox/rox/central/networkgraph/flow/datastore"
	nodeDatastore "github.com/stackrox/rox/central/node/datastore"
	podDatastore "github.com/stackrox/rox/central/pod/datastore"
	"github.com/stackrox/rox/central/postgres"
	processBaselineDatastore "github.com/stackrox/rox/central/processbaseline/datastore"
	processDatastore "github.com/stackrox/rox/central/processindicator/datastore"
	k8sRoleDataStore "github.com/stackrox/rox/central/rbac/k8srole/datastore"
	roleBindingDataStore "github.com/stackrox/rox/central/rbac/k8srolebinding/datastore"
	"github.com/stackrox/rox/central/reports/common"
	snapshotDS "github.com/stackrox/rox/central/reports/snapshot/datastore"
	riskDataStore "github.com/stackrox/rox/central/risk/datastore"
	serviceAccountDataStore "github.com/stackrox/rox/central/serviceaccount/datastore"
	vulnReqDataStore "github.com/stackrox/rox/central/vulnerabilityrequest/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/logging"
	pgPkg "github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/protoutils"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sliceutils"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/timeutil"
	"golang.org/x/sync/semaphore"
)

const (
	pruneInterval      = 1 * time.Hour
	orphanWindow       = 30 * time.Minute
	baselineBatchLimit = 10000
	clusterGCFreq      = 24 * time.Hour
	logImbueGCFreq     = 24 * time.Hour
	logImbueWindow     = 24 * 7 * time.Hour

	alertQueryTimeout = 10 * time.Minute

	flowsSemaphoreWeight = 5
)

var (
	log                   = logging.LoggerForModule()
	pruningCtx            = sac.WithAllAccess(context.Background())
	lastClusterPruneTime  time.Time
	lastLogImbuePruneTime time.Time
)

// GarbageCollector implements a generic garbage collection mechanism.
type GarbageCollector interface {
	Start()
	Stop()
}

func newGarbageCollector(alerts alertDatastore.DataStore,
	nodes nodeDatastore.DataStore,
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
	k8sRoleBindings roleBindingDataStore.DataStore,
	logimbueStore logimbueDataStore.Store,
	reportSnapshotDS snapshotDS.DataStore,
	blobStore blobDatastore.Datastore) GarbageCollector {
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
		logimbueStore:   logimbueStore,
		stopper:         concurrency.NewStopper(),
		postgres:        globaldb.GetPostgres(),
		reportSnapshot:  reportSnapshotDS,
		blobStore:       blobStore,
	}
}

type garbageCollectorImpl struct {
	postgres pgPkg.DB

	alerts          alertDatastore.DataStore
	clusters        clusterDatastore.DataStore
	nodes           nodeDatastore.DataStore
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
	logimbueStore   logimbueDataStore.Store
	stopper         concurrency.Stopper
	reportSnapshot  snapshotDS.DataStore
	blobStore       blobDatastore.Datastore
}

func (g *garbageCollectorImpl) Start() {
	go g.runGC()
}

func (g *garbageCollectorImpl) pruneBasedOnConfig() {
	pvtConfig, err := g.config.GetPrivateConfig(pruningCtx)
	if err != nil {
		log.Error(err)
		return
	}
	if pvtConfig == nil {
		log.Error("UNEXPECTED: Got nil config")
		return
	}
	log.Info("[Pruning] Starting a garbage collection cycle")
	g.collectImages(pvtConfig)
	g.collectAlerts(pvtConfig)
	g.removeOrphanedResources()
	g.removeOrphanedRisks()
	g.removeExpiredVulnRequests()
	g.collectClusters(pvtConfig)
	if features.VulnReportingEnhancements.Enabled() {
		g.removeOldReportHistory(pvtConfig)
		g.removeOldReportBlobs(pvtConfig)
	}
	if features.AdministrationEvents.Enabled() {
		g.removeExpiredAdministrationEvents(pvtConfig)
	}
	postgres.PruneActiveComponents(pruningCtx, g.postgres)
	postgres.PruneClusterHealthStatuses(pruningCtx, g.postgres)

	g.pruneLogImbues()

	log.Info("[Pruning] Finished garbage collection cycle")
}

func (g *garbageCollectorImpl) runGC() {
	defer g.stopper.Flow().ReportStopped()

	lastClusterPruneTime = time.Now().Add(-24 * time.Hour)
	lastLogImbuePruneTime = time.Now().Add(-24 * time.Hour)
	g.pruneBasedOnConfig()

	t := time.NewTicker(pruneInterval)
	for {
		select {
		case <-t.C:
			g.pruneBasedOnConfig()
		case <-g.stopper.Flow().StopRequested():
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
func (g *garbageCollectorImpl) removeOrphanedPods() {
	podIDsToRemove, err := postgres.GetOrphanedPodIDs(pruningCtx, g.postgres)
	if err != nil {
		log.Errorf("Error finding orphaned pods: %v", err)
		return
	}

	if len(podIDsToRemove) == 0 {
		log.Info("[Pruning] Found no orphaned pods...")
		return
	}
	log.Infof("[Pruning] Found %d orphaned pods (from formerly deleted clusters). Deleting...",
		len(podIDsToRemove))

	for _, id := range podIDsToRemove {
		if err := g.pods.RemovePod(pruningCtx, id); err != nil {
			log.Errorf("Failed to remove pod with id %s: %v", id, err)
		}
	}
}

// Remove nodes where the cluster has been deleted.
func (g *garbageCollectorImpl) removeOrphanedNodes() {
	nodesToRemove, err := postgres.GetOrphanedNodeIDs(pruningCtx, g.postgres)
	if err != nil {
		log.Errorf("Error finding orphaned nodes: %v", err)
		return
	}

	if len(nodesToRemove) == 0 {
		log.Info("[Pruning] Found no orphaned nodes...")
		return
	}
	log.Infof("[Pruning] Found %d orphaned nodes (from formerly deleted clusters). Deleting...",
		len(nodesToRemove))

	for _, id := range nodesToRemove {
		if err := g.nodes.DeleteNodes(pruningCtx, id); err != nil {
			log.Errorf("Failed to remove node with id %s: %v", id, err)
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

	g.markOrphanedAlertsAsResolved()
	g.removeOrphanedNetworkFlows(clusterIDSet)

	g.removeOrphanedPods()
	g.removeOrphanedNodes()

	// The deletion of pods can trigger the deletion of indicators.  So in theory there could
	// be fewer indicators to delete if we process orphaned pods first.
	g.removeOrphanedProcesses()
	g.removeOrphanedProcessBaselines(deploymentSet)
	g.removeOrphanedPLOP()

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

func (g *garbageCollectorImpl) removeOrphanedProcesses() {
	postgres.PruneOrphanedProcessIndicators(pruningCtx, g.postgres, orphanWindow)
	log.Info("[Process pruning] Pruning of orphaned processes complete")
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

// removeOrphanedPLOP: cleans up ProcessListeningOnPort objects that do not
// have correct process indicator reference. Such objects could not be cleaned
// on per-deployment basis, since the deployment and cluster info is taken from
// the process indicator, so clean without such filtering in batches.
func (g *garbageCollectorImpl) removeOrphanedPLOP() {
	prunedCount := postgres.PruneOrphanedPLOP(pruningCtx, g.postgres, orphanWindow)

	log.Infof("[PLOP pruning] Found %d orphaned process listening on port objects",
		prunedCount)
}

func (g *garbageCollectorImpl) removeExpiredAdministrationEvents(config *storage.PrivateConfig) {
	retentionDays := time.Duration(config.GetAdministrationEventsConfig().GetRetentionDurationDays()) * 24 * time.Hour
	postgres.PruneAdministrationEvents(pruningCtx, g.postgres, retentionDays)
}

func (g *garbageCollectorImpl) getOrphanedAlerts(ctx context.Context) ([]string, error) {
	return postgres.GetOrphanedAlertIDs(ctx, g.postgres, orphanWindow)
}

func (g *garbageCollectorImpl) markOrphanedAlertsAsResolved() {
	alertsToResolve, err := g.getOrphanedAlerts(pruningCtx)
	if err != nil {
		log.Errorf("[Alert pruning] error getting orphaned alert ids: %v", err)
		return
	}

	log.Infof("[Alert pruning] Found %d orphaned alerts", len(alertsToResolve))
	if _, err := g.alerts.MarkAlertsResolvedBatch(pruningCtx, alertsToResolve...); err != nil {
		log.Error(errors.Wrap(err, "error marking alert as resolved"))
	}
}

func (g *garbageCollectorImpl) removeOrphanedNetworkFlows(clusters set.FrozenStringSet) {
	var wg sync.WaitGroup
	sema := semaphore.NewWeighted(flowsSemaphoreWeight)

	orphanTime := time.Now().UTC().Add(-1 * orphanWindow)

	// Each cluster has a separate store thus we can take advantage of doing these deletions concurrently.  If we don't
	// the entire prune job will be stuck waiting on processing the network flows deletions in cluster sequence.
	for _, c := range clusters.AsSlice() {
		if err := sema.Acquire(pruningCtx, 1); err != nil {
			log.Errorf("context cancelled via stop: %v", err)
			return
		}

		log.Debugf("[Network Flow pruning] for cluster %q", c)
		wg.Add(1)
		go func(c string) {
			defer sema.Release(1)
			defer wg.Done()
			store, err := g.networkflows.GetFlowStore(pruningCtx, c)
			if err != nil {
				log.Errorf("error getting flow store for cluster %q: %v", c, err)
				return
			} else if store == nil {
				return
			}

			err = store.RemoveOrphanedFlows(pruningCtx, &orphanTime)
			if err != nil {
				log.Errorf("error removing orphaned flows for cluster %q: %v", c, err)
			}

			// Second remove stale network flows
			err = store.RemoveStaleFlows(pruningCtx)
			if err != nil {
				log.Errorf("error removing stale flows for cluster %q: %v", c, err)
			}
		}(c)
	}
	wg.Wait()

	log.Info("[Network Flow pruning] Completed")
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

func (g *garbageCollectorImpl) removeOldReportHistory(config *storage.PrivateConfig) {
	reportHistoryRetentionConfig := config.GetReportRetentionConfig().GetHistoryRetentionDurationDays()
	dur := time.Duration(reportHistoryRetentionConfig) * 24 * time.Hour
	postgres.PruneReportHistory(pruningCtx, g.postgres, dur)
}

func (g *garbageCollectorImpl) removeOldReportBlobs(config *storage.PrivateConfig) {
	blobRetentionDays := config.GetReportRetentionConfig().GetDownloadableReportRetentionDays()
	cutOffTime, err := types.TimestampProto(time.Now().Add(-time.Duration(blobRetentionDays) * 24 * time.Hour))
	if err != nil {
		log.Errorf("Failed to determine downloadable report retention %v", err)
		return
	}

	// Sort reversely by modification time
	query := search.NewQueryBuilder().AddRegexes(search.BlobName, common.ReportBlobRegex).WithPagination(
		search.NewPagination().AddSortOption(search.NewSortOption(search.BlobModificationTime).Reversed(true)))
	blobs, err := g.blobStore.SearchMetadata(pruningCtx, query.ProtoQuery())
	if err != nil {
		log.Errorf("Failed to fetch downloadable report metadata: %v", err)
		return
	}

	remainingQuota := int64(config.GetReportRetentionConfig().GetDownloadableReportGlobalRetentionBytes())
	var bytesFreed int64
	var blobsRemoved int
	var toFree bool
	for _, blob := range blobs {
		if !toFree {
			if remainingQuota > blob.GetLength() && blob.GetModifiedTime().Compare(cutOffTime) > 0 {
				remainingQuota = remainingQuota - blob.GetLength()
				continue
			}
			toFree = true
		}
		if err = g.blobStore.Delete(pruningCtx, blob.GetName()); err != nil {
			log.Errorf("Failed to delete blob %+v, will try next time", blob)
			continue
		}
		bytesFreed += blob.GetLength()
		blobsRemoved++
	}
	log.Infof("[Downloadable Report Pruning] Removed %d blobs and freed %d bytes", blobsRemoved, bytesFreed)
}

func (g *garbageCollectorImpl) collectClusters(config *storage.PrivateConfig) {
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
		// In this case, the clusters that became unhealthy after config creation would also be unhealthy for fewer than retention days
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
		log.Debug("[Cluster Pruning] no inactive, non excluded clusters found.")
		return
	}

	for _, clusterID := range clustersToPrune {
		log.Infof("[Cluster Pruning] Removing cluster with ID %s", clusterID)
		if err := g.clusters.RemoveCluster(pruningCtx, clusterID, nil); err != nil {
			log.Error(err)
			return
		}
	}
}

func (g *garbageCollectorImpl) checkIfClusterContainsCentral(cluster *storage.Cluster) (bool, error) {
	// This query could be expensive, but it's a rare occurrence. It only happens if there is a cluster that has been unhealthy for a long time (configurable)
	query := search.NewQueryBuilder().
		AddExactMatches(search.ClusterID, cluster.GetId()).
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
			AddExactMatches(search.LifecycleStage, storage.LifecycleStage_DEPLOY.String()).
			AddExactMatches(search.ViolationState, storage.ViolationState_RESOLVED.String()).
			AddDays(search.ViolationTime, int64(pruneResolvedDeployAfter)).
			ProtoQuery()
		queries = append(queries, q)
	}

	if pruneAllRuntimeAfter > 0 {
		q := search.NewQueryBuilder().
			AddExactMatches(search.LifecycleStage, storage.LifecycleStage_RUNTIME.String()).
			AddDays(search.ViolationTime, int64(pruneAllRuntimeAfter)).
			ProtoQuery()
		queries = append(queries, q)
	}

	if pruneDeletedRuntimeAfter > 0 && pruneAllRuntimeAfter != pruneDeletedRuntimeAfter {
		q := search.NewQueryBuilder().
			AddExactMatches(search.LifecycleStage, storage.LifecycleStage_RUNTIME.String()).
			AddDays(search.ViolationTime, int64(pruneDeletedRuntimeAfter)).
			AddBools(search.Inactive, true).
			ProtoQuery()
		queries = append(queries, q)
	}

	if pruneAttemptedDeployAfter > 0 {
		q := search.NewQueryBuilder().
			AddExactMatches(search.LifecycleStage, storage.LifecycleStage_DEPLOY.String()).
			AddExactMatches(search.ViolationState, storage.ViolationState_ATTEMPTED.String()).
			AddDays(search.ViolationTime, int64(pruneAttemptedDeployAfter)).
			ProtoQuery()
		queries = append(queries, q)
	}

	if pruneAttemptedRuntimeAfter > 0 {
		q := search.NewQueryBuilder().
			AddExactMatches(search.LifecycleStage, storage.LifecycleStage_RUNTIME.String()).
			AddExactMatches(search.ViolationState, storage.ViolationState_ATTEMPTED.String()).
			AddDays(search.ViolationTime, int64(pruneAttemptedRuntimeAfter)).
			ProtoQuery()
		queries = append(queries, q)
	}

	if len(queries) == 0 {
		log.Info("No alert retention configuration, skipping")
		return
	}

	ctx, cancel := context.WithTimeout(pruningCtx, alertQueryTimeout)
	defer cancel()

	alertResults, err := g.alerts.Search(ctx, search.DisjunctionQuery(queries...))
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

func (g *garbageCollectorImpl) pruneLogImbues() {
	// Check to see if enough time has elapsed to run again
	if lastLogImbuePruneTime.Add(logImbueGCFreq).After(time.Now()) {
		// Only log imbue pruning if it's been at least logImbueGCFreq since last time those were pruned
		return
	}
	defer func() {
		lastLogImbuePruneTime = time.Now()
	}()

	postgres.PruneLogImbues(pruningCtx, g.postgres, logImbueWindow)
}

func (g *garbageCollectorImpl) Stop() {
	g.stopper.Client().Stop()
	_ = g.stopper.Client().Stopped().Wait()
}
