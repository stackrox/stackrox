package manager

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	clusterDatastore "github.com/stackrox/rox/central/cluster/datastore"
	"github.com/stackrox/rox/central/compliance"
	"github.com/stackrox/rox/central/compliance/data"
	complianceDS "github.com/stackrox/rox/central/compliance/datastore"
	"github.com/stackrox/rox/central/compliance/framework"
	"github.com/stackrox/rox/central/compliance/standards"
	complianceOperatorCheckDS "github.com/stackrox/rox/central/complianceoperator/checkresults/datastore"
	complianceOperatorManager "github.com/stackrox/rox/central/complianceoperator/manager"
	"github.com/stackrox/rox/central/deployment/datastore"
	nodeDatastore "github.com/stackrox/rox/central/node/datastore"
	podDatastore "github.com/stackrox/rox/central/pod/datastore"
	"github.com/stackrox/rox/central/scrape/factory"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/uuid"
)

const (
	scrapeTimeout = 10 * time.Minute
)

var (
	log = logging.LoggerForModule()

	scrapeDataDeps = []string{"HostScraped"}

	complianceSAC = sac.ForResource(resources.Compliance)
)

type manager struct {
	standardsRegistry         *standards.Registry
	complianceOperatorManager complianceOperatorManager.Manager

	mutex    sync.RWMutex
	runsByID map[string]*runInstance

	stopSig    concurrency.Signal
	interruptC chan struct{}

	clusterStore    clusterDatastore.DataStore
	nodeStore       nodeDatastore.DataStore
	deploymentStore datastore.DataStore
	podStore        podDatastore.DataStore

	dataRepoFactory data.RepositoryFactory
	scrapeFactory   factory.ScrapeFactory

	complianceOperatorResults complianceOperatorCheckDS.DataStore

	resultsStore complianceDS.DataStore
}

func newManager(standardsRegistry *standards.Registry, complianceOperatorManager complianceOperatorManager.Manager, complianceOperatorResults complianceOperatorCheckDS.DataStore, clusterStore clusterDatastore.DataStore, nodeStore nodeDatastore.DataStore, deploymentStore datastore.DataStore, podStore podDatastore.DataStore, dataRepoFactory data.RepositoryFactory, scrapeFactory factory.ScrapeFactory, resultsStore complianceDS.DataStore) *manager {
	mgr := &manager{
		standardsRegistry:         standardsRegistry,
		complianceOperatorManager: complianceOperatorManager,
		complianceOperatorResults: complianceOperatorResults,

		runsByID: make(map[string]*runInstance),

		interruptC: make(chan struct{}),

		clusterStore:    clusterStore,
		nodeStore:       nodeStore,
		deploymentStore: deploymentStore,
		podStore:        podStore,

		dataRepoFactory: dataRepoFactory,
		scrapeFactory:   scrapeFactory,

		resultsStore: resultsStore,
	}
	return mgr
}

func (m *manager) createDomain(ctx context.Context, clusterID string) (framework.ComplianceDomain, error) {
	cluster, ok, err := m.clusterStore.GetCluster(ctx, clusterID)
	if err == nil && !ok {
		err = errors.New("cluster not found")
	}
	if err != nil {
		return nil, errors.Wrapf(err, "could not get cluster with ID %q", clusterID)
	}

	nodes, err := m.nodeStore.SearchRawNodes(ctx, search.NewQueryBuilder().AddExactMatches(search.ClusterID, clusterID).ProtoQuery())
	if err != nil {
		return nil, errors.Wrapf(err, "retrieving nodes for cluster %s", clusterID)
	}

	query := search.NewQueryBuilder().AddExactMatches(search.ClusterID, clusterID).ProtoQuery()
	deployments, err := m.deploymentStore.SearchRawDeployments(ctx, query)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get deployments for cluster %s", clusterID)
	}

	query = search.NewQueryBuilder().AddExactMatches(search.ClusterID, clusterID).ProtoQuery()
	pods, err := m.podStore.SearchRawPods(ctx, query)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get pods for cluster %s", clusterID)
	}

	machineConfigs, err := m.complianceOperatorManager.GetMachineConfigs(clusterID)
	if err != nil {
		return nil, errors.Wrapf(err, "getting machine configs for cluster %s", clusterID)
	}
	return framework.NewComplianceDomain(cluster, nodes, deployments, pods, machineConfigs), nil
}

func (m *manager) createRun(domain framework.ComplianceDomain, standard *standards.Standard) *runInstance {
	runID := uuid.NewV4().String()
	run := createRun(runID, domain, standard)
	concurrency.WithLock(&m.mutex, func() {
		m.runsByID[runID] = run
	})
	return run
}

func (m *manager) interrupt() {
	select {
	case <-m.stopSig.Done():
	case m.interruptC <- struct{}{}:
	}
}

func runMatches(request *v1.GetRecentComplianceRunsRequest, runProto *v1.ComplianceRun) bool {
	if request.GetSince() != nil && runProto.GetStartTime().Compare(request.GetSince()) < 0 {
		return false
	}
	if request.GetClusterIdOpt() != nil && runProto.GetClusterId() != request.GetClusterId() {
		return false
	}
	if request.GetStandardIdOpt() != nil && runProto.GetStandardId() != request.GetStandardId() {
		return false
	}
	return true
}

func (m *manager) GetRecentRuns(ctx context.Context, request *v1.GetRecentComplianceRunsRequest) ([]*v1.ComplianceRun, error) {
	runs := m.getRuns(request)

	// Filter out runs the user does not have read access to.
	var returnedRuns []*v1.ComplianceRun
	for _, run := range runs {
		if ok, err := complianceSAC.ReadAllowed(ctx, sac.ClusterScopeKey(run.GetClusterId())); err != nil {
			return nil, err
		} else if ok {
			returnedRuns = append(returnedRuns, run)
		}
	}

	return returnedRuns, nil
}

func (m *manager) getRuns(request *v1.GetRecentComplianceRunsRequest) []*v1.ComplianceRun {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	var result []*v1.ComplianceRun
	for _, run := range m.runsByID {
		runProto := run.ToProto()
		if !runMatches(request, runProto) {
			continue
		}
		result = append(result, runProto)
	}
	return result
}

func (m *manager) GetRecentRun(ctx context.Context, id string) (*v1.ComplianceRun, error) {
	run := m.getRun(id)
	if run == nil {
		return nil, fmt.Errorf("run with id %q not found", id)
	}

	// Check read access to the cluster the run is for.
	if ok, err := complianceSAC.ReadAllowed(ctx, sac.ClusterScopeKey(run.ClusterId)); err != nil {
		return nil, err
	} else if !ok {
		return nil, errox.NotFound
	}

	return run, nil
}

func (m *manager) getRun(id string) *v1.ComplianceRun {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	run := m.runsByID[id]
	if run == nil {
		return nil
	}
	return run.ToProto()
}

func (m *manager) ExpandSelection(ctx context.Context, clusterIDOrWildcard, standardIDOrWildcard string) ([]compliance.ClusterStandardPair, error) {
	clusterIDs, err := m.expandClusters(ctx, clusterIDOrWildcard)
	if err != nil {
		return nil, err
	}
	standardIDs := m.expandStandards(standardIDOrWildcard)

	result := make([]compliance.ClusterStandardPair, 0, len(clusterIDs)*len(standardIDs))
	for _, clusterID := range clusterIDs {
		for _, standardID := range standardIDs {
			result = append(result, compliance.ClusterStandardPair{
				ClusterID:  clusterID,
				StandardID: standardID,
			})
		}
	}
	return result, nil
}

func (m *manager) expandClusters(ctx context.Context, clusterIDOrWildcard string) ([]string, error) {
	if clusterIDOrWildcard != Wildcard {
		return []string{clusterIDOrWildcard}, nil
	}

	clusters, err := m.clusterStore.GetClusters(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "retrieving clusters")
	}
	var clusterIDs []string
	for _, cluster := range clusters {
		if complianceSAC.ScopeChecker(ctx, storage.Access_READ_WRITE_ACCESS).IsAllowed(sac.ClusterScopeKey(cluster.GetId())) {
			clusterIDs = append(clusterIDs, cluster.GetId())
		}
	}
	return clusterIDs, nil
}

func (m *manager) expandStandards(standardIDOrWildcard string) []string {
	if standardIDOrWildcard != Wildcard {
		return []string{standardIDOrWildcard}
	}

	allStandards := m.standardsRegistry.AllStandards()
	standardIDs := make([]string, 0, len(allStandards))
	for _, standard := range allStandards {
		standardIDs = append(standardIDs, standard.ID)
	}
	return standardIDs
}

func (m *manager) TriggerRuns(ctx context.Context, clusterStandardPairs ...compliance.ClusterStandardPair) ([]*v1.ComplianceRun, error) {
	// Trigger the runs.
	runs, err := m.createAndLaunchRuns(ctx, clusterStandardPairs)
	if err != nil {
		return nil, err
	}

	runProtos := make([]*v1.ComplianceRun, len(runs))
	for i, run := range runs {
		runProtos[i] = run.ToProto()
	}
	return runProtos, nil
}

func (m *manager) createAndLaunchRuns(ctx context.Context, clusterStandardPairs []compliance.ClusterStandardPair) ([]*runInstance, error) {
	// Check write access to all of the runs
	clusterScopes := newClusterScopeCollector()
	for _, clusterStandardPair := range clusterStandardPairs {
		clusterScopes.add(clusterStandardPair.ClusterID)
	}
	if !complianceSAC.ScopeChecker(ctx, storage.Access_READ_WRITE_ACCESS).AllAllowed(clusterScopes.get()) {
		return nil, sac.ErrResourceAccessDenied
	}

	// If successful, elevate privileges to read all data needed.
	// Input Context needs to live beyond request, so use background as the underlying context instead of the one passed in.
	elevatedCtx := sac.WithAllAccess(context.Background())

	// Step 1: Group all standard implementations that need to run by cluster ID.
	standardsByClusterID := make(map[string][]*standards.Standard)
	for _, clusterAndStandard := range clusterStandardPairs {
		standard := m.standardsRegistry.LookupStandard(clusterAndStandard.StandardID)
		if standard == nil {
			return nil, fmt.Errorf("invalid compliance standard ID %q", clusterAndStandard.StandardID)
		}
		standardsByClusterID[clusterAndStandard.ClusterID] = append(standardsByClusterID[clusterAndStandard.ClusterID], standard)
	}

	var runs []*runInstance
	// Step 2: For each cluster, instantiate domains and scrape promises, and create runs.
	for clusterID, standardImpls := range standardsByClusterID {
		domain, err := m.createDomain(elevatedCtx, clusterID)
		if err != nil {
			return nil, errors.Wrapf(err, "could not create domain for cluster ID %q", clusterID)
		}
		domainPB := getDomainProto(domain)
		// Domain is indirectly scoped, and checks global permissions for write operations.
		// Temporarily elevating the privileges to write the domain informations.
		domainWriteCtx := sac.WithGlobalAccessScopeChecker(
			ctx,
			sac.AllowFixedScopes(
				sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
				sac.ResourceScopeKeys(resources.Compliance),
			),
		)
		err = m.resultsStore.StoreComplianceDomain(domainWriteCtx, domainPB)
		if err != nil {
			return nil, errors.Wrapf(err, "could not create domain protobuf for ID %q", clusterID)
		}

		var scrapeBasedPromise, scrapeLessPromise dataPromise
		for _, standard := range standardImpls {
			if !m.complianceOperatorManager.IsStandardActiveForCluster(standard.ID, clusterID) {
				continue
			}
			run := m.createRun(domain, standard)
			var dataPromise dataPromise
			if standard.HasAnyDataDependency(scrapeDataDeps...) {
				if scrapeBasedPromise == nil {
					var standardIDs []string
					for _, standard := range standardImpls {
						standardIDs = append(standardIDs, standard.ID)
					}
					scrapeBasedPromise = createAndRunScrape(elevatedCtx, m.scrapeFactory, m.dataRepoFactory, domain, scrapeTimeout, standardIDs)
				}
				dataPromise = scrapeBasedPromise
			} else {
				if scrapeLessPromise == nil {
					scrapeLessPromise = newFixedDataPromise(elevatedCtx, m.dataRepoFactory, domain)
				}
				dataPromise = scrapeLessPromise
			}

			run.Start(dataPromise, m.resultsStore)
			runs = append(runs, run)
		}
	}

	m.interrupt()

	return runs, nil
}

func (m *manager) GetRunStatuses(ctx context.Context, ids ...string) ([]*v1.ComplianceRun, error) {
	runStatuses := m.getRunStatuses(ids)

	// Check read access to all of the runs
	clusterScopes := newClusterScopeCollector()
	for _, runStatus := range runStatuses {
		clusterScopes.add(runStatus.GetClusterId())
	}
	if !complianceSAC.ScopeChecker(ctx, storage.Access_READ_ACCESS).AllAllowed(clusterScopes.get()) {
		return nil, errox.NotFound
	}

	return runStatuses, nil
}

func (m *manager) getRunStatuses(ids []string) []*v1.ComplianceRun {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	var result []*v1.ComplianceRun
	for _, id := range ids {
		if run := m.runsByID[id]; run != nil {
			result = append(result, run.ToProto())
		}
	}
	return result
}

// Helper class for collecting a set of cluster scopes to perform sac checks against.
func newClusterScopeCollector() *clusterScopeCollector {
	return &clusterScopeCollector{
		clusterIDs: set.NewStringSet(),
		scopeKeys:  make([][]sac.ScopeKey, 0),
	}
}

type clusterScopeCollector struct {
	clusterIDs set.StringSet
	scopeKeys  [][]sac.ScopeKey
}

func (csc *clusterScopeCollector) add(clusterID string) {
	if !csc.clusterIDs.Contains(clusterID) {
		csc.clusterIDs.Add(clusterID)
		csc.scopeKeys = append(csc.scopeKeys, []sac.ScopeKey{sac.ClusterScopeKey(clusterID)})
	}
}

func (csc *clusterScopeCollector) get() [][]sac.ScopeKey {
	return csc.scopeKeys
}
