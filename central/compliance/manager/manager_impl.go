package manager

import (
	"errors"
	"fmt"
	"sync"
	"time"

	clusterDatastore "github.com/stackrox/rox/central/cluster/datastore"
	"github.com/stackrox/rox/central/compliance/framework"
	"github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/node/store"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/timeutil"
	"github.com/stackrox/rox/pkg/uuid"
)

var (
	log = logging.LoggerForModule()
)

type manager struct {
	scheduleStore ScheduleStore

	mutex         sync.RWMutex
	runsByID      map[string]*runInstance
	schedulesByID map[string]*scheduleInstance

	stopSig    concurrency.Signal
	interruptC chan struct{}

	clusterStore    clusterDatastore.DataStore
	nodeStore       store.GlobalStore
	deploymentStore datastore.DataStore
}

func newManager(scheduleStore ScheduleStore, clusterStore clusterDatastore.DataStore, nodeStore store.GlobalStore, deploymentStore datastore.DataStore) (*manager, error) {
	mgr := &manager{
		scheduleStore: scheduleStore,
		runsByID:      make(map[string]*runInstance),
		schedulesByID: make(map[string]*scheduleInstance),

		interruptC: make(chan struct{}),

		clusterStore:    clusterStore,
		nodeStore:       nodeStore,
		deploymentStore: deploymentStore,
	}

	if err := mgr.readFromStore(); err != nil {
		return nil, fmt.Errorf("reading schedules from store: %v", err)
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

func (m *manager) createDomain(clusterID string) (framework.ComplianceDomain, error) {
	cluster, ok, err := m.clusterStore.GetCluster(clusterID)
	if err == nil && !ok {
		err = errors.New("cluster not found")
	}
	if err != nil {
		return nil, fmt.Errorf("could not get cluster with ID %q: %v", clusterID, err)
	}

	clusterNodeStore, err := m.nodeStore.GetClusterNodeStore(clusterID)
	if err != nil {
		return nil, fmt.Errorf("could not get node store for cluster %s: %v", clusterID, err)
	}
	nodes, err := clusterNodeStore.ListNodes()
	if err != nil {
		return nil, fmt.Errorf("listing nodes for cluster %s: %v", clusterID, err)
	}

	query := search.NewQueryBuilder().AddStrings(search.ClusterID, clusterID).ProtoQuery()
	deployments, err := m.deploymentStore.SearchRawDeployments(query)
	if err != nil {
		return nil, fmt.Errorf("could not get deployments for cluster %s: %v", clusterID, err)
	}

	return framework.NewComplianceDomain(cluster, nodes, deployments), nil
}

func (m *manager) createRun(clusterID, standardID string) (*runInstance, error) {
	domain, err := m.createDomain(clusterID)
	if err != nil {
		return nil, fmt.Errorf("creating compliance domain: %v", err)
	}

	allChecks, err := checksForStandard(standardID)
	if err != nil {
		return nil, fmt.Errorf("looking up checks for standard %q: %v", standardID, err)
	}

	complianceRun, err := framework.NewComplianceRun(allChecks...)
	if err != nil {
		return nil, fmt.Errorf("instantiating compliance run: %v", err)
	}

	runID := uuid.NewV4().String()
	run, err := createRun(runID, standardID, domain, complianceRun)
	if err != nil {
		return nil, err
	}

	return run, nil
}

func (m *manager) startRun(run *runInstance) error {
	if err := run.Start(); err != nil {
		return err
	}

	concurrency.WithLock(&m.mutex, func() {
		m.runsByID[run.id] = run
	})
	return nil
}

func (m *manager) createRunFromSchedule(schedule *scheduleInstance) (*runInstance, error) {
	r, err := m.createRun(schedule.clusterAndStandard())
	if err != nil {
		return nil, fmt.Errorf("creating run: %v", err)
	}
	r.schedule = schedule
	return r, nil
}

func (m *manager) runSchedules(schedulesToRun []*scheduleInstance) error {
	var errList errorhelpers.ErrorList
	for _, sched := range schedulesToRun {
		run, err := m.createRunFromSchedule(sched)
		if err != nil {
			errList.AddStringf("creating compliance run: %v", err)
			continue
		}
		if err := m.startRun(run); err != nil {
			errList.AddStringf("starting compliance run: %v", err)
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
	for _, runID := range runsToDelete.AsSlice() {
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
		m.runSchedules(schedulesToRun)
	}

	nextRunTime := m.nearestRunTime()

	var nextRunTimer *time.Timer
	if !nextRunTime.IsZero() {
		nextRunTimer = time.NewTimer(nextRunTime.Sub(time.Now()))
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

func (m *manager) DeleteSchedule(id string) error {
	var err error
	concurrency.WithLock(&m.mutex, func() {
		_, ok := m.schedulesByID[id]
		if !ok {
			err = fmt.Errorf("schedule with ID %q not found", id)
			return
		}

		delete(m.schedulesByID, id)
	})
	if err != nil {
		return err
	}

	// No need to interrupt - the next run time can only shift further into the future.

	if err := m.scheduleStore.DeleteSchedule(id); err != nil {
		return fmt.Errorf("deleting schedule from store: %v", err)
	}

	return nil
}

func (m *manager) AddSchedule(spec *storage.ComplianceRunSchedule) (*v1.ComplianceRunScheduleInfo, error) {
	if spec.GetId() != "" {
		return nil, errors.New("schedule to add must have an empty ID")
	}

	if _, ok, err := m.clusterStore.GetCluster(spec.GetClusterId()); !ok || err != nil {
		if err == nil {
			err = errors.New("no such cluster")
		}
		return nil, fmt.Errorf("could not check cluster ID %q: %v", spec.GetClusterId(), err)
	}

	if _, err := checksForStandard(spec.GetStandardId()); err != nil {
		return nil, fmt.Errorf("invalid standard ID %q: %v", spec.GetStandardId(), err)
	}

	spec.Id = uuid.NewV4().String()

	scheduleMD, err := newScheduleInstance(spec)
	if err != nil {
		return nil, fmt.Errorf("instantiating schedule: %v", err)
	}

	concurrency.WithLock(&m.mutex, func() {
		m.schedulesByID[spec.Id] = scheduleMD
	})

	m.interrupt()
	return scheduleMD.ToProto(), nil
}

func (m *manager) UpdateSchedule(spec *storage.ComplianceRunSchedule) (*v1.ComplianceRunScheduleInfo, error) {
	if spec.GetId() == "" {
		return nil, errors.New("schedule to update must have a non-empty ID")
	}

	if _, ok, err := m.clusterStore.GetCluster(spec.GetClusterId()); !ok || err != nil {
		if err == nil {
			err = errors.New("no such cluster")
		}
		return nil, fmt.Errorf("could not check cluster ID %q: %v", spec.GetClusterId(), err)
	}

	if _, err := checksForStandard(spec.GetStandardId()); err != nil {
		return nil, fmt.Errorf("invalid standard ID %q: %v", spec.GetStandardId(), err)
	}

	var scheduleInstance *scheduleInstance
	concurrency.WithRLock(&m.mutex, func() {
		scheduleInstance = m.schedulesByID[spec.GetId()]
	})
	if scheduleInstance == nil {
		return nil, fmt.Errorf("no schedule with id %q", spec.GetId())
	}

	if err := scheduleInstance.update(spec); err != nil {
		return nil, fmt.Errorf("could not update schedule %s: %v", spec.GetId(), err)
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

func (m *manager) GetSchedules(request *v1.GetComplianceRunSchedulesRequest) []*v1.ComplianceRunScheduleInfo {
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

func (m *manager) GetSchedule(id string) (*v1.ComplianceRunScheduleInfo, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	schedule := m.schedulesByID[id]
	if schedule == nil {
		return nil, fmt.Errorf("schedule with id %q not found", id)
	}
	return schedule.ToProto(), nil
}

func runMatches(request *v1.GetRecentComplianceRunsRequest, runProto *v1.ComplianceRun) bool {
	if request.GetSince() != nil && protoconv.CompareProtoTimestamps(runProto.GetStartTime(), request.GetSince()) < 0 {
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

func (m *manager) GetRecentRuns(request *v1.GetRecentComplianceRunsRequest) []*v1.ComplianceRun {
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

func (m *manager) GetRecentRun(id string) (*v1.ComplianceRun, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	run := m.runsByID[id]
	if run == nil {
		return nil, fmt.Errorf("run with id %q not found", id)
	}
	return run.ToProto(), nil
}

func (m *manager) TriggerRun(clusterID, standardID string) (*v1.ComplianceRun, error) {
	run, err := m.createRun(clusterID, standardID)
	if err != nil {
		return nil, fmt.Errorf("creating run: %v", err)
	}
	if err := m.startRun(run); err != nil {
		return nil, fmt.Errorf("starting run: %v", err)
	}

	m.interrupt()

	return run.ToProto(), nil
}
