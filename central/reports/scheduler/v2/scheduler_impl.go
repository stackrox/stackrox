package v2

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"
	notifierDS "github.com/stackrox/rox/central/notifier/datastore"
	"github.com/stackrox/rox/central/reports/common"
	reportConfigDS "github.com/stackrox/rox/central/reports/config/datastore"
	reportGen "github.com/stackrox/rox/central/reports/scheduler/v2/reportgenerator"
	"github.com/stackrox/rox/central/reports/scheduler/v2/reportqueue"
	reportSnapshotDS "github.com/stackrox/rox/central/reports/snapshot/datastore"
	"github.com/stackrox/rox/central/reports/validation"
	collectionDS "github.com/stackrox/rox/central/resourcecollection/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/dblock"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/protoconv/schedule"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/sync"
	"golang.org/x/sync/semaphore"
	"gopkg.in/robfig/cron.v2"
)

var log = logging.LoggerForModule()

// queueGeneratorBinding pairs a report job queue with the generator that processes
// jobs from that queue. E.g., image CVE report job queue <-> image CVE report
// generator, node CVE report job queue <-> node CVE report generator.
type queueGeneratorBinding struct {
	queue     *reportqueue.ReportQueue
	generator reportGen.ReportGenerator
}

type scheduler struct {
	// Used to map reportConfigs to their cron jobs. This is only used for scheduled reports, On-demand reports are directly added to reportsQueue
	reportConfigToEntryIDs map[string]cron.EntryID

	reportConfigDatastore reportConfigDS.DataStore
	reportSnapshotStore   reportSnapshotDS.DataStore
	collectionDatastore   collectionDS.DataStore
	notifierDatastore     notifierDS.DataStore
	validator             *validation.Validator

	// Ordered list of queue-generator bindings. The main loop round-robins through
	// these when selecting the next report to run.
	queues       []queueGeneratorBinding
	nextQueueIdx int
	// Maps report type to its queue for routing submissions.
	queueByType map[storage.ReportSnapshot_ReportType]*reportqueue.ReportQueue

	// Signals the scheduler to find and run a new report. Triggered when a new
	// request is enqueued or when a report completes. Reset when no runnable report
	// is found across any queue.
	readyForReports concurrency.Signal

	/* Concurrency and synchronization related fields */
	// isStarted permits submissions to enter the local queues after lock acquisition.
	isStarted atomic.Bool
	// isStopped will prevent scheduler from being re-started once it is stopped
	isStopped atomic.Bool

	// lifecycleLock serializes Start and Stop, including Stop before Start.
	lifecycleLock  sync.Mutex
	cronStop       sync.Once
	startRequested bool
	ctx            context.Context
	cancel         context.CancelCauseFunc
	done           chan struct{}
	ready          atomic.Bool

	// Use to synchronize access to reportConfigToEntryIDs map
	cronJobsLock sync.Mutex
	// Use to lock any database tables if needed to prevent race conditions
	dbLock sync.Mutex
	// NOTE: Lock only one mutex at a time. Do not lock another mutex when one is already held.
	//      If you need to lock another mutex, you must free the locked one first.

	cron            *cron.Cron
	concurrencySema *semaphore.Weighted
}

// New instantiates a new cron scheduler and supports adding and removing report requests
func New(reportConfigDatastore reportConfigDS.DataStore, reportSnapshotStore reportSnapshotDS.DataStore,
	collectionDatastore collectionDS.DataStore, notifierDatastore notifierDS.DataStore,
	imageReportGenerator reportGen.ReportGenerator, nodeReportGenerator reportGen.ReportGenerator,
	validator *validation.Validator) Scheduler {

	cronScheduler := cron.New()
	cronScheduler.Start()
	return newSchedulerImpl(reportConfigDatastore, reportSnapshotStore, collectionDatastore, notifierDatastore,
		imageReportGenerator, nodeReportGenerator, validator, cronScheduler)
}

func newSchedulerImpl(reportConfigDatastore reportConfigDS.DataStore, reportSnapshotStore reportSnapshotDS.DataStore,
	collectionDatastore collectionDS.DataStore, notifierDatastore notifierDS.DataStore,
	imageReportGenerator reportGen.ReportGenerator, nodeReportGenerator reportGen.ReportGenerator,
	validator *validation.Validator, cronScheduler *cron.Cron) *scheduler {

	imageQueue := reportqueue.New()
	queues := []queueGeneratorBinding{
		{queue: imageQueue, generator: imageReportGenerator},
	}
	queueByType := map[storage.ReportSnapshot_ReportType]*reportqueue.ReportQueue{
		storage.ReportSnapshot_VULNERABILITY: imageQueue,
	}

	if features.NodeVulnerabilityReports.Enabled() && nodeReportGenerator != nil {
		nodeQueue := reportqueue.New()
		queues = append(queues, queueGeneratorBinding{queue: nodeQueue, generator: nodeReportGenerator})
		queueByType[storage.ReportSnapshot_NODE_VULNERABILITY] = nodeQueue
	}

	ctx, cancel := context.WithCancelCause(context.Background())
	s := &scheduler{
		reportConfigToEntryIDs: make(map[string]cron.EntryID),
		reportConfigDatastore:  reportConfigDatastore,
		reportSnapshotStore:    reportSnapshotStore,
		collectionDatastore:    collectionDatastore,
		notifierDatastore:      notifierDatastore,
		validator:              validator,
		queues:                 queues,
		nextQueueIdx:           0,
		queueByType:            queueByType,
		readyForReports:        concurrency.NewSignal(),
		ctx:                    ctx,
		cancel:                 cancel,
		done:                   make(chan struct{}),
		cron:                   cronScheduler,
		concurrencySema:        semaphore.NewWeighted(int64(env.ReportExecutionMaxConcurrency.IntegerSetting())),
	}
	return s
}

/* Concurrency and scheduling functions */

// Start begins acquiring ownership in the background. Contention and transient
// database errors are retried until acquisition succeeds or Stop is called.
func (s *scheduler) Start(db postgres.DB) {
	s.lifecycleLock.Lock()
	defer s.lifecycleLock.Unlock()
	if s.startRequested || s.isStopped.Load() {
		return
	}
	s.startRequested = true
	go s.run(db)
}

// Ready reports whether this scheduler owns the lock and has initialized its queues.
func (s *scheduler) Ready() bool {
	return s.ready.Load() && !s.isStopped.Load()
}

func (s *scheduler) run(db postgres.DB) {
	defer close(s.done)
	defer s.isStarted.Store(false)
	defer s.ready.Store(false)

	for s.ctx.Err() == nil {
		acquired, release, err := dblock.TryAcquireAdvisoryLock(s.ctx, db, dblock.ReportSchedulerLockID)
		if acquired {
			// Release only after execution and all in-flight reports have stopped.
			defer release()
			if s.ctx.Err() != nil {
				return
			}
			s.isStarted.Store(true)
			// Retry each initialization phase separately, so later query failures do
			// not enqueue previously recovered reports a second time.
			for _, initialize := range []func() error{s.queuePendingReports, s.recoverMissedSchedules, s.queueScheduledReports} {
				if err := s.retryInitialization(initialize); err != nil {
					return
				}
			}
			if s.ctx.Err() != nil {
				return
			}
			s.ready.Store(true)
			log.Info("Report scheduler acquired ownership and is ready")
			s.runReports()
			return
		}
		if err != nil {
			log.Errorf("Report scheduler: failed to acquire advisory lock, retrying: %v", err)
		} else {
			log.Info("Report scheduler: advisory lock held by another process, retrying")
		}
		select {
		case <-s.ctx.Done():
			return
		case <-time.After(5 * time.Second):
		}
	}
}

// Stop cancels acquisition and execution, waits for reports to stop, and releases ownership.
// It is safe to call concurrently or before Start, and a stopped scheduler cannot restart.
func (s *scheduler) Stop() {
	started := concurrency.WithLock1(&s.lifecycleLock, func() bool {
		s.isStopped.Store(true)
		s.cancel(reportGen.ErrSchedulerStopped)
		return s.startRequested
	})
	if started {
		<-s.done
	}
	s.cronStop.Do(func() {
		s.cronJobsLock.Lock()
		defer s.cronJobsLock.Unlock()
		s.cron.Stop()
	})
}

func (s *scheduler) runReports() {
	var running sync.WaitGroup
	defer running.Wait()
	for {
		select {
		case <-s.ctx.Done():
			return
		case <-s.readyForReports.Done():
			req, q, gen := s.selectNextJobRoundRobin()
			if req == nil {
				s.readyForReports.Reset()
				continue
			}
			if err := s.concurrencySema.Acquire(s.ctx, 1); err != nil {
				return
			}
			if s.ctx.Err() != nil {
				s.concurrencySema.Release(1)
				return
			}
			log.Infof("Executing report '%s' at %v", req.ReportSnapshot.GetName(), time.Now().Format(time.RFC822))
			running.Go(func() {
				s.runSingleReport(q, gen, req)
			})
		}
	}
}

func (s *scheduler) selectNextJobRoundRobin() (*reportGen.ReportRequest, *reportqueue.ReportQueue, reportGen.ReportGenerator) {
	n := len(s.queues)
	for i := range n {
		idx := (s.nextQueueIdx + i) % n
		qb := s.queues[idx]
		if req := qb.queue.Dequeue(); req != nil {
			s.nextQueueIdx = (idx + 1) % n
			return req, qb.queue, qb.generator
		}
	}
	return nil, nil, nil
}

func (s *scheduler) runSingleReport(q *reportqueue.ReportQueue, gen reportGen.ReportGenerator, req *reportGen.ReportRequest) {
	defer s.readyForReports.Signal()
	defer s.concurrencySema.Release(1)
	defer q.MarkReportDoneForConfig(req.ReportSnapshot.GetReportConfigurationId())

	reportID := req.ReportSnapshot.GetReportId()
	ctx, cancel := context.WithCancelCause(s.ctx)
	q.AddCancelFunc(reportID, cancel)
	defer cancel(nil)
	defer q.RemoveCancelFunc(reportID)

	gen.ProcessReportRequest(ctx, req)
}

// UpsertReportSchedule adds/updates the schedule at which reports for the given report config are executed.
func (s *scheduler) UpsertReportSchedule(reportConfig *storage.ReportConfiguration) error {
	s.cronJobsLock.Lock()
	defer s.cronJobsLock.Unlock()

	if s.isStopped.Load() {
		return nil
	}

	// Remove the old entry if this is an update
	if oldEntryID, ok := s.reportConfigToEntryIDs[reportConfig.GetId()]; ok {
		s.cron.Remove(oldEntryID)
	}
	if reportConfig.GetSchedule() != nil {
		cronSpec, err := schedule.ConvertToCronTab(reportConfig.GetSchedule())
		if err != nil {
			return err
		}
		entryID, err := s.cron.AddFunc(cronSpec, s.reportClosure(reportConfig))
		if err != nil {
			return err
		}
		s.reportConfigToEntryIDs[reportConfig.GetId()] = entryID
	}
	return nil
}

// RemoveReportSchedule removes the given report configuration from scheduled execution.
func (s *scheduler) RemoveReportSchedule(reportConfigID string) {
	s.cronJobsLock.Lock()
	defer s.cronJobsLock.Unlock()

	oldEntryID, exists := s.reportConfigToEntryIDs[reportConfigID]
	if exists {
		s.cron.Remove(oldEntryID)
		delete(s.reportConfigToEntryIDs, reportConfigID)
	}
}

func (s *scheduler) GetScheduledConfigIDs() []string {
	s.cronJobsLock.Lock()
	defer s.cronJobsLock.Unlock()

	ids := make([]string, 0, len(s.reportConfigToEntryIDs))
	for id := range s.reportConfigToEntryIDs {
		ids = append(ids, id)
	}
	return ids
}

/* Functions to add/remove report jobs from queue */

// CancelReportRequest cancels a report request. If the report is waiting in queue, it is removed
// and its snapshot is updated to FAILURE with a cancellation message. If the report is already
// being prepared, its context is cancelled, which propagates cancellation to in-flight database
// queries and blob store writes.
func (s *scheduler) CancelReportRequest(ctx context.Context, reportID string) (bool, error) {
	for _, qb := range s.queues {
		req := qb.queue.Remove(reportID)
		if req != nil {
			req.ReportSnapshot.ReportStatus.ErrorMsg = reportGen.ErrUserCancelled.Error()
			req.ReportSnapshot.ReportStatus.CompletedAt = protocompat.TimestampNow()
			req.ReportSnapshot.ReportStatus.RunState = storage.ReportStatus_FAILURE
			if err := s.reportSnapshotStore.UpdateReportSnapshot(ctx, req.ReportSnapshot); err != nil {
				return false, errors.Wrapf(err, "Error updating report snapshot to FAILURE for report ID '%s'", reportID)
			}
			return true, nil
		}
	}
	for _, qb := range s.queues {
		if qb.queue.TryCancel(reportID) {
			return true, nil
		}
	}
	return false, nil
}

func (s *scheduler) CanSubmitReportRequest(user *storage.SlimUser, reportConfig *storage.ReportConfiguration) (bool, error) {
	return s.doesUserHavePendingReport(reportConfig.GetId(), user.GetId(),
		storage.ReportSnapshot_ReportType(reportConfig.GetType()))
}

// SubmitReportRequest submits a report execution request. The report request can be either for an on demand report or a scheduled report.
// If there is already a pending report request submitted by the same user for the same report config, this request will be denied.
// However, there can be multiple pending report requests for same configuration by different users.
func (s *scheduler) SubmitReportRequest(ctx context.Context, request *reportGen.ReportRequest, reSubmission bool) (string, error) {
	err := reportGen.ValidateReportRequest(request)
	if err != nil {
		return "", err
	}

	q := s.queueForSnapshot(request.ReportSnapshot)
	if q == nil {
		return "", errors.New("node vulnerability reports are not enabled")
	}

	request.ReportSnapshot.ReportStatus.RunState = storage.ReportStatus_WAITING
	request.ReportSnapshot.ReportStatus.QueuedAt = protocompat.TimestampNow()
	reportID, err := s.validateAndPersistSnapshot(ctx, request.ReportSnapshot, reSubmission)
	if err != nil {
		return "", err
	}

	request.ReportSnapshot.ReportId = reportID

	if s.isStarted.Load() && !s.isStopped.Load() {
		q.Enqueue(request)
		s.readyForReports.Signal()
	}

	return request.ReportSnapshot.GetReportId(), nil
}

func (s *scheduler) queueForSnapshot(snap *storage.ReportSnapshot) *reportqueue.ReportQueue {
	return s.queueByType[snap.GetType()]
}

func (s *scheduler) reportClosure(reportConfig *storage.ReportConfiguration) func() {
	return func() {
		log.Infof("Submitting scheduled report request for '%s' at %v", reportConfig.GetName(), time.Now().Format(time.RFC850))
		reportReq, err := s.validator.ValidateAndGenerateReportRequest(reportConfig.GetId(), storage.ReportStatus_EMAIL,
			storage.ReportStatus_SCHEDULED, nil)
		if err != nil {
			log.Errorf("Error submitting scheduled report request for '%s': %s", reportConfig.GetName(), err)
		}
		_, err = s.SubmitReportRequest(sac.WithAllAccess(s.ctx), reportReq, false)
		if err != nil {
			log.Errorf("Error submitting scheduled report request for '%s': %s", reportConfig.GetName(), err)
		}
	}
}

// retryInitialization retries a single discovery query or record without repeating
// successfully queued records. Cancellation also interrupts the retry delay.
func (s *scheduler) retryInitialization(initialize func() error) error {
	for s.ctx.Err() == nil {
		if err := initialize(); err == nil {
			return nil
		} else {
			log.Errorf("Report scheduler initialization failed, retrying: %v", err)
		}
		select {
		case <-s.ctx.Done():
			return s.ctx.Err()
		case <-time.After(5 * time.Second):
		}
	}
	return s.ctx.Err()
}

func (s *scheduler) queuePendingReports() error {
	pendingReportsQuery := search.NewQueryBuilder().
		AddExactMatches(search.ReportState, storage.ReportStatus_WAITING.String(), storage.ReportStatus_PREPARING.String()).
		WithPagination(search.NewPagination().AddSortOption(search.NewSortOption(search.ReportQueuedTime))).
		ProtoQuery()
	pendingReports, err := s.reportSnapshotStore.SearchReportSnapshots(sac.WithAllAccess(s.ctx), pendingReportsQuery)
	if err != nil {
		return errors.Wrap(err, "finding pending reports")
	}
	for _, snap := range pendingReports {
		if err := s.retryInitialization(func() error { return s.queuePendingReport(snap) }); err != nil {
			return err
		}
	}
	return nil
}

func (s *scheduler) queuePendingReport(snap *storage.ReportSnapshot) error {
	req := &reportGen.ReportRequest{ReportSnapshot: snap}
	// View-based reports have no configuration or collection to reload.
	if snap.GetReportStatus().GetReportRequestType() != storage.ReportStatus_VIEW_BASED {
		_, found, err := s.reportConfigDatastore.GetReportConfiguration(sac.WithAllAccess(s.ctx), snap.GetReportConfigurationId())
		if err != nil {
			return errors.Wrapf(err, "loading report configuration %s", snap.GetReportConfigurationId())
		}
		if !found {
			log.Errorf("Report configuration %s no longer exists; skipping pending report %s", snap.GetReportConfigurationId(), snap.GetReportId())
			return nil
		}
		if !common.HasValidResourceScope(snap.GetResourceScope()) {
			log.Errorf("Report configuration %s has an empty resource scope; skipping pending report %s", snap.GetReportConfigurationId(), snap.GetReportId())
			return nil
		}
		if snap.GetCollection() != nil {
			collection, found, err := s.collectionDatastore.Get(sac.WithAllAccess(s.ctx), snap.GetCollection().GetId())
			if err != nil {
				return errors.Wrapf(err, "loading collection %s", snap.GetCollection().GetId())
			}
			if !found {
				log.Errorf("Collection %s no longer exists; skipping pending report %s", snap.GetCollection().GetId(), snap.GetReportId())
				return nil
			}
			req.Collection = collection
		}
	}
	// Invalid/disabled report types cannot be fixed by retrying a database operation.
	if err := reportGen.ValidateReportRequest(req); err != nil {
		log.Errorf("Skipping invalid pending report %s: %v", snap.GetReportId(), err)
		return nil
	}
	if s.queueForSnapshot(snap) == nil {
		return nil
	}
	_, err := s.SubmitReportRequest(sac.WithAllAccess(s.ctx), req, true)
	return err
}

func (s *scheduler) queueScheduledReports() error {
	reportTypes := []string{storage.ReportConfiguration_VULNERABILITY.String()}
	if features.NodeVulnerabilityReports.Enabled() {
		reportTypes = append(reportTypes, storage.ReportConfiguration_NODE_VULNERABILITY.String())
	}
	query := search.NewQueryBuilder().
		AddExactMatches(search.ReportType, reportTypes...).
		ProtoQuery()
	reportConfigs, err := s.reportConfigDatastore.GetReportConfigurations(sac.WithAllAccess(s.ctx), query)
	if err != nil {
		return errors.Wrap(err, "finding scheduled reports")
	}
	for _, rc := range reportConfigs {
		if !common.HasValidResourceScope(rc.GetResourceScope()) {
			log.Errorf("Skipping scheduled report for config '%s' (ID: %s): resource scope is empty",
				rc.GetName(), rc.GetId())
			continue
		}
		if rc.GetSchedule() != nil {
			if err := s.UpsertReportSchedule(rc); err != nil {
				log.Errorf("Error queuing scheduled report for report configuration with ID %s: %v", rc.GetId(), err)
			}
		}
	}
	return nil
}

func (s *scheduler) recoverMissedSchedules() error {
	if !env.ReportMissedScheduleRecovery.BooleanSetting() {
		return nil
	}

	reportTypes := []string{storage.ReportConfiguration_VULNERABILITY.String()}
	if features.NodeVulnerabilityReports.Enabled() {
		reportTypes = append(reportTypes, storage.ReportConfiguration_NODE_VULNERABILITY.String())
	}
	query := search.NewQueryBuilder().
		AddExactMatches(search.ReportType, reportTypes...).
		ProtoQuery()
	reportConfigs, err := s.reportConfigDatastore.GetReportConfigurations(sac.WithAllAccess(s.ctx), query)
	if err != nil {
		return errors.Wrap(err, "finding report configs for missed schedule recovery")
	}

	for _, rc := range reportConfigs {
		if rc.GetSchedule() == nil {
			continue
		}

		cronSpec, err := schedule.ConvertToCronTab(rc.GetSchedule())
		if err != nil {
			log.Errorf("Error converting schedule to crontab for config '%s': %v", rc.GetId(), err)
			continue
		}

		cronSchedule, err := cron.Parse(cronSpec)
		if err != nil {
			log.Errorf("Error parsing cron spec for config '%s': %v", rc.GetId(), err)
			continue
		}

		// Find the most recent time this schedule should have fired.
		// We approximate the previous fire time by stepping back from now.
		now := time.Now()
		previousFireTime := findPreviousFireTime(cronSchedule, now)
		if previousFireTime.IsZero() {
			continue
		}

		// Query for the most recent report snapshot for this config
		snapshotQuery := search.NewQueryBuilder().
			AddExactMatches(search.ReportConfigID, rc.GetId()).
			AddExactMatches(search.ReportRequestType, storage.ReportStatus_SCHEDULED.String()).
			WithPagination(search.NewPagination().
				AddSortOption(search.NewSortOption(search.ReportQueuedTime).Reversed(true)).
				Limit(1)).
			ProtoQuery()
		snapshots, err := s.reportSnapshotStore.SearchReportSnapshots(sac.WithAllAccess(s.ctx), snapshotQuery)
		if err != nil {
			log.Errorf("Error querying snapshots for missed schedule recovery, config '%s': %v", rc.GetId(), err)
			continue
		}

		// If there are no snapshots, the schedule has never run yet. Don't recover it;
		// the cron job will fire at the correct time.
		if len(snapshots) == 0 {
			continue
		}

		lastSnapshot := snapshots[0]
		// If the most recent snapshot is still pending, queuePendingReports already handles it.
		runState := lastSnapshot.GetReportStatus().GetRunState()
		if runState == storage.ReportStatus_WAITING || runState == storage.ReportStatus_PREPARING {
			continue
		}

		shouldRecover := false
		lastTime := lastSnapshot.GetReportStatus().GetQueuedAt()
		if lastTime != nil {
			lastQueuedAt, err := protocompat.ConvertTimestampToTimeOrError(lastTime)
			if err != nil {
				log.Errorf("Error converting timestamp for config '%s': %v", rc.GetId(), err)
				continue
			}
			if lastQueuedAt.Before(previousFireTime) {
				shouldRecover = true
			}
		}

		if shouldRecover {
			log.Infof("Recovering missed scheduled report for config '%s' (name: '%s')", rc.GetId(), rc.GetName())
			reportReq, err := s.validator.ValidateAndGenerateReportRequest(rc.GetId(), storage.ReportStatus_EMAIL,
				storage.ReportStatus_SCHEDULED, nil)
			if err != nil {
				log.Errorf("Error generating report request for missed schedule recovery, config '%s': %v", rc.GetId(), err)
				continue
			}
			_, err = s.SubmitReportRequest(sac.WithAllAccess(s.ctx), reportReq, false)
			if err != nil {
				log.Errorf("Error submitting missed scheduled report for config '%s': %v", rc.GetId(), err)
			}
		}
	}
	return nil
}

// findPreviousFireTime finds the most recent time before `now` that the cron schedule would have fired.
// It does this by stepping back in time and checking when the next fire time from that point would be.
func findPreviousFireTime(cronSchedule cron.Schedule, now time.Time) time.Time {
	// Start from 32 days ago to cover monthly schedules (max interval between fires)
	candidate := now.Add(-32 * 24 * time.Hour)
	var previousFire time.Time
	for {
		next := cronSchedule.Next(candidate)
		if next.After(now) {
			break
		}
		previousFire = next
		candidate = next
	}
	return previousFire
}

// Validate report snapshot and store it to db if validation succeeds.
// Will return report_id if successful.
// Validation will check if the user requesting the report doesn't already have a pending report for the same config
func (s *scheduler) validateAndPersistSnapshot(ctx context.Context, snapshot *storage.ReportSnapshot, reSubmission bool) (string, error) {
	s.dbLock.Lock()
	defer s.dbLock.Unlock()
	var err error
	if !reSubmission {
		requestType := snapshot.GetReportStatus().GetReportRequestType()
		reportType := snapshot.GetType()

		if requestType == storage.ReportStatus_ON_DEMAND {
			userHasAnotherReport, err := s.doesUserHavePendingReport(snapshot.GetReportConfigurationId(), snapshot.GetRequester().GetId(), reportType)
			if err != nil {
				return "", err
			}
			if userHasAnotherReport {
				return "", errors.Wrapf(errox.AlreadyExists, "User already has a report running for config ID '%s'",
					snapshot.GetReportConfigurationId())
			}
		}

		if requestType == storage.ReportStatus_VIEW_BASED {
			userHasAnotherReport, err := s.doesUserHaveViewBasedPendingReport(snapshot.GetRequester().GetId(), reportType)
			if err != nil {
				return "", err
			}
			if userHasAnotherReport {
				return "", errors.New("User already has a view based report queued")
			}
		}

		// View-based reports are authorized at the gRPC layer with only view permissions
		// (no WorkflowAdministration write required). The snapshot datastore requires
		// WorkflowAdministration write, so we elevate to a privileged context here.
		persistCtx := ctx
		if requestType == storage.ReportStatus_VIEW_BASED {
			persistCtx = sac.WithAllAccess(ctx)
		}
		snapshot.ReportId, err = s.reportSnapshotStore.AddReportSnapshot(persistCtx, snapshot)
	} else {
		err = s.reportSnapshotStore.UpdateReportSnapshot(ctx, snapshot)
	}

	if err != nil {
		return "", err
	}
	return snapshot.GetReportId(), nil
}

func (s *scheduler) doesUserHaveViewBasedPendingReport(userID string, reportType storage.ReportSnapshot_ReportType) (bool, error) {
	query := search.NewQueryBuilder().
		AddExactMatches(search.ReportState, storage.ReportStatus_WAITING.String(), storage.ReportStatus_PREPARING.String()).
		AddExactMatches(search.ReportRequestType, storage.ReportStatus_VIEW_BASED.String()).
		AddExactMatches(search.UserID, userID).
		AddExactMatches(search.ReportType, reportType.String()).
		ProtoQuery()
	runningReports, err := s.reportSnapshotStore.Count(sac.WithAllAccess(s.ctx), query)
	if err != nil {
		return false, err
	}
	return runningReports > 0, nil
}

func (s *scheduler) doesUserHavePendingReport(configID string, userID string, reportType storage.ReportSnapshot_ReportType) (bool, error) {
	query := search.NewQueryBuilder().
		AddExactMatches(search.ReportConfigID, configID).
		AddExactMatches(search.ReportState, storage.ReportStatus_WAITING.String(), storage.ReportStatus_PREPARING.String()).
		AddExactMatches(search.ReportRequestType, storage.ReportStatus_ON_DEMAND.String()).
		AddExactMatches(search.ReportType, reportType.String()).
		ProtoQuery()
	runningReports, err := s.reportSnapshotStore.SearchReportSnapshots(sac.WithAllAccess(s.ctx), query)
	if err != nil {
		return false, err
	}
	for _, rep := range runningReports {
		if rep.GetRequester().GetId() == userID {
			return true, nil
		}
	}
	return false, nil
}
