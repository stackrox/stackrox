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
	nodeDatastore "github.com/stackrox/rox/central/node/globaldatastore"
	podDatastore "github.com/stackrox/rox/central/pod/datastore"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/central/scrape/factory"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/bolthelper"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/timeutil"
	"github.com/stackrox/rox/pkg/uuid"
)

const (
	scrapeTimeout = 10 * time.Minute
)

var (
	log = logging.LoggerForModule()

	scrapeDataDeps = []string{"HostScraped"}

	complianceRunSAC         = sac.ForResource(resources.ComplianceRuns)
	complianceRunScheduleSAC = sac.ForResource(resources.ComplianceRunSchedule)
)

type manager struct {
	scheduleStore             ScheduleStore
	standardsRegistry         *standards.Registry
	complianceOperatorManager complianceOperatorManager.Manager

	mutex         sync.RWMutex
	runsByID      map[string]*runInstance
	schedulesByID map[string]*scheduleInstance

	stopSig    concurrency.Signal
	interruptC chan struct{}

	clusterStore    clusterDatastore.DataStore
	nodeStore       nodeDatastore.GlobalDataStore
	deploymentStore datastore.DataStore
	podStore        podDatastore.DataStore

	dataRepoFactory data.RepositoryFactory
	scrapeFactory   factory.ScrapeFactory

	complianceOperatorResults complianceOperatorCheckDS.DataStore

	resultsStore complianceDS.DataStore
}

func newManager(standardsRegistry *standards.Registry, complianceOperatorManager complianceOperatorManager.Manager, complianceOperatorResults complianceOperatorCheckDS.DataStore, scheduleStore ScheduleStore, clusterStore clusterDatastore.DataStore, nodeStore nodeDatastore.GlobalDataStore, deploymentStore datastore.DataStore, podStore podDatastore.DataStore, dataRepoFactory data.RepositoryFactory, scrapeFactory factory.ScrapeFactory, resultsStore complianceDS.DataStore) (*manager, error) {
	mgr := &manager{
		scheduleStore:             scheduleStore,
		standardsRegistry:         standardsRegistry,
		complianceOperatorManager: complianceOperatorManager,
		complianceOperatorResults: complianceOperatorResults,

		runsByID:      make(map[string]*runInstance),
		schedulesByID: make(map[string]*scheduleInstance),

		interruptC: make(chan struct{}),

		clusterStore:    clusterStore,
		nodeStore:       nodeStore,
		deploymentStore: deploymentStore,
		podStore:        podStore,

		dataRepoFactory: dataRepoFactory,
		scrapeFactory:   scrapeFactory,

		resultsStore: resultsStore,
	}

	if err := mgr.readFromStore(); err != nil {
		return nil, errors.Wrap(err, "reading schedules from store")
	}
	return mgr, nil
}

func (m *manager) readFromStore() error {
	scheduleProtos, err := m.scheduleStore.ListSchedules()
	if err != nil {
		return err
	}

	for _, scheduleProto := range scheduleProtos {
		scheduleInstance, err := newScheduleInstance(scheduleProto)
		if err != nil {
			log.Errorf("Could not instantiate stored schedule: %v", err)
			continue
		}
		m.schedulesByID[scheduleProto.GetId()] = scheduleInstance
	}

	return nil
}

func (m *manager) Start() error {
	if !m.stopSig.Reset() {
		return errors.New("compliance manager is already running")
	}
	go m.run()
	return nil
}

func (m *manager) Stop() error {
	if !m.stopSig.Signal() {
		return errors.New("compliance manager was not running")
	}
	return nil
}

func (m *manager) createDomain(ctx context.Context, clusterID string) (framework.ComplianceDomain, error) {
	cluster, ok, err := m.clusterStore.GetCluster(ctx, clusterID)
	if err == nil && !ok {
		err = errors.New("cluster not found")
	}
	if err != nil {
		return nil, errors.Wrapf(err, "could not get cluster with ID %q", clusterID)
	}

	clusterNodeStore, err := m.nodeStore.GetClusterNodeStore(ctx, clusterID, false)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get node store for cluster %s", clusterID)
	}
	nodes, err := clusterNodeStore.ListNodes()

	if errors.Cause(err) == bolthelper.ErrBucketNotFound {
		nodes = nil
	} else if err != nil {
		return nil, errors.Wrapf(err, "listing nodes for cluster %s", clusterID)
	}

	query := search.NewQueryBuilder().AddStrings(search.ClusterID, clusterID).ProtoQuery()
	deployments, err := m.deploymentStore.SearchRawDeployments(ctx, query)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get deployments for cluster %s", clusterID)
	}

	query = search.NewQueryBuilder().AddStrings(search.ClusterID, clusterID).ProtoQuery()
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

func (m *manager) createRun(domain framework.ComplianceDomain, standard *standards.Standard, schedule *scheduleInstance) *runInstance {
	runID := uuid.NewV4().String()
	run := createRun(runID, domain, standard)
	run.schedule = schedule
	concurrency.WithLock(&m.mutex, func() {
		m.runsByID[runID] = run
	})
	return run
}

func (m *manager) createRunFromSchedule(ctx context.Context, schedule *scheduleInstance) ([]*runInstance, error) {
	return m.createAndLaunchRuns(ctx, []compliance.ClusterStandardPair{schedule.clusterAndStandard()}, schedule)
}

func (m *manager) runSchedules(ctx context.Context, schedulesToRun []*scheduleInstance) error {
	var errList errorhelpers.ErrorList
	for _, sched := range schedulesToRun {
		_, err := m.createRunFromSchedule(ctx, sched)
		if err != nil {
			errList.AddStringf("creating compliance run: %v", err)
			continue
		}
	}
	return errList.ToError()
}

func (m *manager) schedulesToRun() []*scheduleInstance {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	now := time.Now()

	var result []*scheduleInstance

	for _, schedule := range m.schedulesByID {
		if schedule.checkAndUpdate(now) {
			result = append(result, schedule)
		}
	}

	return result
}

func (m *manager) nearestRunTime() time.Time {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	var nearestTime time.Time

	for _, schedule := range m.schedulesByID {
		if schedule.nextRunTime.IsZero() {
			continue
		}
		if nearestTime.IsZero() || schedule.nextRunTime.Before(nearestTime) {
			nearestTime = schedule.nextRunTime
		}
	}

	return nearestTime
}

func (m *manager) cleanupRuns() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Step 1: Gather runs marked for deletion.
	runsToDelete := set.NewStringSet()
	for id, run := range m.runsByID {
		if run.shouldDelete() {
			runsToDelete.Add(id)
		}
	}

	// Step 2: Preserve all runs which are referenced by a schedule instance
	for _, schedule := range m.schedulesByID {
		concurrency.WithRLock(&schedule.mutex, func() {
			if schedule.lastRun != nil {
				runsToDelete.Remove(schedule.lastRun.id)
			}
			if schedule.lastFinishedRun != nil {
				runsToDelete.Remove(schedule.lastFinishedRun.id)
			}
		})
	}

	// Step 3: Actually delete the runs
	for runID := range runsToDelete {
		delete(m.runsByID, runID)
	}
}

func (m *manager) runLoopSingle() {
	// Clean up runs in the loop. This might mean that runs stick around for way longer than 12h, but this doesn't
	// really matter as we are guaranteed to run this loop whenever we would accumulate new runs, so the number of runs
	// we store is still bounded.
	m.cleanupRuns()

	schedulesToRun := m.schedulesToRun()
	if len(schedulesToRun) > 0 {
		runComplianceCtx := sac.WithAllAccess(context.Background())
		if err := m.runSchedules(runComplianceCtx, schedulesToRun); err != nil {
			log.Errorf("Failed to run schedules: %v", err)
		}
	}

	nextRunTime := m.nearestRunTime()

	var nextRunTimer *time.Timer
	if !nextRunTime.IsZero() {
		nextRunTimer = time.NewTimer(time.Until(nextRunTime))
	}

	select {
	case <-m.stopSig.Done():
	case <-timeutil.TimerC(nextRunTimer):
		nextRunTimer = nil
	case <-m.interruptC:
	}

	timeutil.StopTimer(nextRunTimer)
}

func (m *manager) run() {
	defer m.stopSig.Signal()

	for !m.stopSig.IsDone() {
		m.runLoopSingle()
	}
}

func (m *manager) interrupt() {
	select {
	case <-m.stopSig.Done():
	case m.interruptC <- struct{}{}:
	}
}

func (m *manager) DeleteSchedule(ctx context.Context, id string) error {
	// Look up the schedule.
	var scheduleProto *v1.ComplianceRunScheduleInfo
	var err error
	concurrency.WithLock(&m.mutex, func() {
		schedule, ok := m.schedulesByID[id]
		if !ok {
			err = fmt.Errorf("schedule with ID %q not found", id)
			return
		}
		scheduleProto = schedule.ToProto()
	})
	if err != nil {
		return err
	}

	// Check write access to the cluster the schedule is for.
	if ok, err := complianceRunScheduleSAC.WriteAllowed(ctx, sac.ClusterScopeKey(scheduleProto.GetSchedule().GetClusterId())); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}

	// No need to interrupt - the next run time can only shift further into the future.
	concurrency.WithLock(&m.mutex, func() {
		delete(m.schedulesByID, id)
	})
	if err := m.scheduleStore.DeleteSchedule(id); err != nil {
		return errors.Wrap(err, "deleting schedule from store")
	}
	return nil
}

func (m *manager) AddSchedule(ctx context.Context, spec *storage.ComplianceRunSchedule) (*v1.ComplianceRunScheduleInfo, error) {
	if spec.GetId() != "" {
		return nil, errors.New("schedule to add must have an empty ID")
	}

	// Check write access to the cluster the schedule is for.
	if ok, err := complianceRunScheduleSAC.WriteAllowed(ctx, sac.ClusterScopeKey(spec.GetClusterId())); err != nil {
		return nil, err
	} else if !ok {
		return nil, sac.ErrResourceAccessDenied
	}

	if ok, err := m.clusterStore.Exists(ctx, spec.GetClusterId()); !ok || err != nil {
		if err == nil {
			err = errors.New("no such cluster")
		}
		return nil, errors.Wrapf(err, "could not check cluster ID %q", spec.GetClusterId())
	}

	if standard := m.standardsRegistry.LookupStandard(spec.GetStandardId()); standard == nil {
		return nil, fmt.Errorf("invalid standard ID %q", spec.GetStandardId())
	}

	spec.Id = uuid.NewV4().String()

	scheduleMD, err := newScheduleInstance(spec)
	if err != nil {
		return nil, errors.Wrap(err, "instantiating schedule")
	}

	concurrency.WithLock(&m.mutex, func() {
		m.schedulesByID[spec.Id] = scheduleMD
	})

	m.interrupt()
	return scheduleMD.ToProto(), nil
}

func (m *manager) UpdateSchedule(ctx context.Context, spec *storage.ComplianceRunSchedule) (*v1.ComplianceRunScheduleInfo, error) {
	if spec.GetId() == "" {
		return nil, errors.New("schedule to update must have a non-empty ID")
	}

	// Check write access to the cluster the schedule is for.
	if ok, err := complianceRunScheduleSAC.WriteAllowed(ctx, sac.ClusterScopeKey(spec.GetClusterId())); err != nil {
		return nil, err
	} else if !ok {
		return nil, sac.ErrResourceAccessDenied
	}

	if ok, err := m.clusterStore.Exists(ctx, spec.GetClusterId()); !ok || err != nil {
		if err == nil {
			err = errors.New("no such cluster")
		}
		return nil, errors.Wrapf(err, "could not check cluster ID %q", spec.GetClusterId())
	}

	if standard := m.standardsRegistry.LookupStandard(spec.GetStandardId()); standard == nil {
		return nil, fmt.Errorf("invalid standard ID %q", spec.GetStandardId())
	}

	var scheduleInstance *scheduleInstance
	concurrency.WithRLock(&m.mutex, func() {
		scheduleInstance = m.schedulesByID[spec.GetId()]
	})
	if scheduleInstance == nil {
		return nil, fmt.Errorf("no schedule with id %q", spec.GetId())
	}

	if err := scheduleInstance.update(spec); err != nil {
		return nil, errors.Wrapf(err, "could not update schedule %s", spec.GetId())
	}

	m.interrupt()
	return scheduleInstance.ToProto(), nil
}

func scheduleMatches(req *v1.GetComplianceRunSchedulesRequest, scheduleProto *storage.ComplianceRunSchedule) bool {
	if req.GetClusterIdOpt() != nil && scheduleProto.GetClusterId() != req.GetClusterId() {
		return false
	}
	if req.GetStandardIdOpt() != nil && scheduleProto.GetStandardId() != req.GetStandardId() {
		return false
	}
	if req.GetSuspendedOpt() != nil && scheduleProto.GetSuspended() != req.GetSuspended() {
		return false
	}
	return true
}

func (m *manager) GetSchedules(ctx context.Context, request *v1.GetComplianceRunSchedulesRequest) ([]*v1.ComplianceRunScheduleInfo, error) {
	schedules := m.getSchedules(request)

	// Check read access to all of the runs
	// Filter out runs the user does not have read access to.
	var returnedSchedules []*v1.ComplianceRunScheduleInfo
	for _, schedule := range schedules {
		if ok, err := complianceRunSAC.ReadAllowed(ctx, sac.ClusterScopeKey(schedule.GetSchedule().GetClusterId())); err != nil {
			return nil, err
		} else if ok {
			returnedSchedules = append(returnedSchedules, schedule)
		}
	}

	return returnedSchedules, nil
}

func (m *manager) getSchedules(request *v1.GetComplianceRunSchedulesRequest) []*v1.ComplianceRunScheduleInfo {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	var result []*v1.ComplianceRunScheduleInfo
	for _, schedule := range m.schedulesByID {
		scheduleInfoProto := schedule.ToProto()
		if scheduleMatches(request, scheduleInfoProto.GetSchedule()) {
			result = append(result, scheduleInfoProto)
		}
	}
	return result
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
		if ok, err := complianceRunSAC.ReadAllowed(ctx, sac.ClusterScopeKey(run.GetClusterId())); err != nil {
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
	if ok, err := complianceRunSAC.ReadAllowed(ctx, sac.ClusterScopeKey(run.ClusterId)); err != nil {
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
		if complianceRunSAC.ScopeChecker(ctx, storage.Access_READ_WRITE_ACCESS).IsAllowed(sac.ClusterScopeKey(cluster.GetId())) {
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
	runs, err := m.createAndLaunchRuns(ctx, clusterStandardPairs, nil)
	if err != nil {
		return nil, err
	}

	runProtos := make([]*v1.ComplianceRun, len(runs))
	for i, run := range runs {
		runProtos[i] = run.ToProto()
	}
	return runProtos, nil
}

func (m *manager) createAndLaunchRuns(ctx context.Context, clusterStandardPairs []compliance.ClusterStandardPair, schedule *scheduleInstance) ([]*runInstance, error) {
	// Check write access to all of the runs
	clusterScopes := newClusterScopeCollector()
	for _, clusterStandardPair := range clusterStandardPairs {
		clusterScopes.add(clusterStandardPair.ClusterID)
	}
	if !complianceRunSAC.ScopeChecker(ctx, storage.Access_READ_WRITE_ACCESS).AllAllowed(clusterScopes.get()) {
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
		err = m.resultsStore.StoreComplianceDomain(ctx, domainPB)
		if err != nil {
			return nil, errors.Wrapf(err, "could not create domain protobuf for ID %q", clusterID)
		}

		var scrapeBasedPromise, scrapeLessPromise dataPromise
		for _, standard := range standardImpls {
			if !m.complianceOperatorManager.IsStandardActiveForCluster(standard.ID, clusterID) {
				continue
			}
			run := m.createRun(domain, standard, schedule)
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
	if !complianceRunSAC.ScopeChecker(ctx, storage.Access_READ_ACCESS).AllAllowed(clusterScopes.get()) {
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
