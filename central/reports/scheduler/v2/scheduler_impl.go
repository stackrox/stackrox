package v2

import (
	"container/list"
	"context"
	"sync/atomic"
	"time"

	"github.com/gogo/protobuf/types"
	"github.com/graph-gophers/graphql-go"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/graphql/resolvers"
	notifierDS "github.com/stackrox/rox/central/notifier/datastore"
	"github.com/stackrox/rox/central/reports/common"
	reportConfigDS "github.com/stackrox/rox/central/reports/config/datastore"
	reportGen "github.com/stackrox/rox/central/reports/scheduler/v2/reportgenerator"
	reportSnapshotDS "github.com/stackrox/rox/central/reports/snapshot/datastore"
	"github.com/stackrox/rox/central/reports/validation"
	collectionDS "github.com/stackrox/rox/central/resourcecollection/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/protoconv/schedule"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sync"
	"golang.org/x/sync/semaphore"
	"gopkg.in/robfig/cron.v2"
)

var (
	log = logging.LoggerForModule()

	scheduledCtx = sac.WithAllAccess(context.Background())
)

type scheduler struct {
	// Used to map reportConfigs to their cron jobs. This is only used for scheduled reports, On-demand reports are directly added to reportsQueue
	reportConfigToEntryIDs map[string]cron.EntryID

	reportConfigDatastore reportConfigDS.DataStore
	reportSnapshotStore   reportSnapshotDS.DataStore
	collectionDatastore   collectionDS.DataStore
	notifierDatastore     notifierDS.DataStore
	reportGenerator       reportGen.ReportGenerator
	validator             *validation.Validator

	reportRequestsQueue *list.List

	// Use to signal the scheduler to find and run a new report if a routine is available
	// This signal is triggered when a new request is added to reportsQueue. It is also triggered when a report completes
	// execution to inform the scheduler that a routine is free. The signal is reset when there is no report to run.
	readyForReports concurrency.Signal
	// Stores config IDs for which a report is currently running. Used to make sure only one report per config runs at a time.
	runningReportConfigs set.StringSet
	Schema               *graphql.Schema

	/* Concurrency and synchronization related fields */
	// isStarted will make sure only one scheduling routine runs for an instance of scheduler
	isStarted atomic.Bool
	// isStopped will prevent scheduler from being re-started once it is stopped
	isStopped atomic.Bool

	/* Concurrency and synchronization related fields */
	stopper concurrency.Stopper

	// Use to synchronize access to reportConfigToEntryIDs map
	cronJobsLock sync.Mutex
	// Use to synchronize access to reportsQueue and runningReportConfigs
	schedulerLock sync.Mutex
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
	reportGenerator reportGen.ReportGenerator, validator *validation.Validator) Scheduler {

	cronScheduler := cron.New()
	cronScheduler.Start()
	ourSchema, err := graphql.ParseSchema(resolvers.Schema(), resolvers.New())
	if err != nil {
		panic(err)
	}
	return newSchedulerImpl(reportConfigDatastore, reportSnapshotStore, collectionDatastore, notifierDatastore,
		reportGenerator, validator, cronScheduler, ourSchema)
}

func newSchedulerImpl(reportConfigDatastore reportConfigDS.DataStore, reportSnapshotStore reportSnapshotDS.DataStore,
	collectionDatastore collectionDS.DataStore, notifierDatastore notifierDS.DataStore,
	reportGenerator reportGen.ReportGenerator, validator *validation.Validator, cronScheduler *cron.Cron,
	schema *graphql.Schema) *scheduler {
	s := &scheduler{
		reportConfigToEntryIDs: make(map[string]cron.EntryID),
		reportConfigDatastore:  reportConfigDatastore,
		reportSnapshotStore:    reportSnapshotStore,
		collectionDatastore:    collectionDatastore,
		notifierDatastore:      notifierDatastore,
		reportGenerator:        reportGenerator,
		validator:              validator,
		reportRequestsQueue:    list.New(),
		readyForReports:        concurrency.NewSignal(),
		runningReportConfigs:   set.NewStringSet(),
		Schema:                 schema,
		stopper:                concurrency.NewStopper(),
		cron:                   cronScheduler,
		concurrencySema:        semaphore.NewWeighted(int64(env.ReportExecutionMaxConcurrency.IntegerSetting())),
	}
	return s
}

/* Concurrency and scheduling functions */

// Start scheduler. A scheduler instance can only be started once. It cannot be re-started once stopped.
// This func will log errors if the scheduler fails to start.
func (s *scheduler) Start() {
	if s.isStopped.Load() {
		log.Error("Scheduler already stopped. It cannot be re-started once stopped")
		return
	}
	swapped := s.isStarted.CompareAndSwap(false, true)
	if !swapped {
		log.Error("Scheduler already running")
		return
	}
	s.queuePendingReports()
	s.queueScheduledReports()
	go s.runReports()
}

// Stop scheduler
func (s *scheduler) Stop() {
	if !s.isStarted.Load() {
		log.Error("Scheduler not started")
		return
	}
	swapped := s.isStopped.CompareAndSwap(false, true)
	if !swapped {
		log.Error("Scheduler already stopped")
		return
	}
	s.stopper.Client().Stop()
	err := s.stopper.Client().Stopped().Wait()
	if err != nil {
		log.Errorf("Error stopping vulnerability report scheduler : %v", err)
	}
}

func (s *scheduler) runReports() {
	defer s.stopper.Flow().ReportStopped()
	for {
		select {
		case <-s.stopper.Flow().StopRequested():
			return
		case <-s.readyForReports.Done():
			reportRequest := s.selectNextRunnableReport()
			if reportRequest == nil {
				s.readyForReports.Reset()
				continue
			}
			if err := s.concurrencySema.Acquire(scheduledCtx, 1); err != nil {
				log.Errorf("Error acquiring semaphore to run new report: %v", err)
				continue
			}
			log.Infof("Executing report '%s' at %v", reportRequest.ReportSnapshot.GetName(), time.Now().Format(time.RFC822))
			go s.runSingleReport(reportRequest)
		}
	}
}

func (s *scheduler) selectNextRunnableReport() *reportGen.ReportRequest {
	s.schedulerLock.Lock()
	defer s.schedulerLock.Unlock()

	request := findAndRemoveFromQueue(s.reportRequestsQueue, func(req *reportGen.ReportRequest) bool {
		return !s.runningReportConfigs.Contains(req.ReportSnapshot.GetReportConfigurationId())
	})
	if request == nil {
		return nil
	}
	s.runningReportConfigs.Add(request.ReportSnapshot.GetReportConfigurationId())
	return request
}

func (s *scheduler) runSingleReport(req *reportGen.ReportRequest) {
	defer s.readyForReports.Signal()
	defer s.concurrencySema.Release(1)
	defer s.removeFromRunningReportConfigs(req.ReportSnapshot.GetReportConfigurationId())

	s.reportGenerator.ProcessReportRequest(req)
}

func (s *scheduler) removeFromRunningReportConfigs(configID string) {
	s.schedulerLock.Lock()
	defer s.schedulerLock.Unlock()
	s.runningReportConfigs.Remove(configID)
}

// UpsertReportSchedule adds/updates the schedule at which reports for the given report config are executed.
func (s *scheduler) UpsertReportSchedule(reportConfig *storage.ReportConfiguration) error {
	s.cronJobsLock.Lock()
	defer s.cronJobsLock.Unlock()

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

/* Functions to add/remove report jobs from queue */

// CancelReportRequest cancels a report request that is still waiting in queue. A user can only cancel a report requested by them.
// If the report is already being prepared or has completed execution, it cannot be cancelled.
func (s *scheduler) CancelReportRequest(ctx context.Context, reportID string) (bool, error) {
	removed := s.tryRemoveFromRequestQueue(reportID)
	if !removed {
		return false, nil
	}
	err := s.reportSnapshotStore.DeleteReportSnapshot(ctx, reportID)
	if err != nil {
		return false, errors.Wrapf(err, "Error deleting report ID '%s' from storage", reportID)
	}
	return true, nil
}

// Returns true if the request was successfully removed from the ReportRequests queue
func (s *scheduler) tryRemoveFromRequestQueue(reportID string) bool {
	s.schedulerLock.Lock()
	defer s.schedulerLock.Unlock()

	request := findAndRemoveFromQueue(s.reportRequestsQueue, func(req *reportGen.ReportRequest) bool {
		return req.ReportSnapshot.GetReportId() == reportID
	})
	return request != nil
}

func (s *scheduler) CanSubmitReportRequest(user *storage.SlimUser, reportConfig *storage.ReportConfiguration) (bool, error) {
	return s.doesUserHavePendingReport(reportConfig.GetId(), user.GetId())
}

// SubmitReportRequest submits a report execution request. The report request can be either for an on demand report or a scheduled report.
// If there is already a pending report request submitted by the same user for the same report config, this request will be denied.
// However, there can be multiple pending report requests for same configuration by different users.
func (s *scheduler) SubmitReportRequest(ctx context.Context, request *reportGen.ReportRequest, reSubmission bool) (string, error) {
	err := reportGen.ValidateReportRequest(request)
	if err != nil {
		return "", err
	}

	request.ReportSnapshot.ReportStatus.RunState = storage.ReportStatus_WAITING
	request.ReportSnapshot.ReportStatus.QueuedAt = types.TimestampNow()
	request.ReportSnapshot.ReportId, err = s.validateAndPersistSnapshot(ctx, request.ReportSnapshot, reSubmission)
	if err != nil {
		return "", err
	}

	s.appendToReportsQueue(request)

	return request.ReportSnapshot.GetReportId(), nil
}

func (s *scheduler) appendToReportsQueue(req *reportGen.ReportRequest) {
	s.schedulerLock.Lock()
	defer s.schedulerLock.Unlock()
	s.reportRequestsQueue.PushBack(req)
	s.readyForReports.Signal()
}

func (s *scheduler) reportClosure(reportConfig *storage.ReportConfiguration) func() {
	return func() {
		log.Infof("Submitting scheduled report request for '%s' at %v", reportConfig.GetName(), time.Now().Format(time.RFC850))
		reportReq, err := s.validator.ValidateAndGenerateReportRequest(reportConfig.GetId(), storage.ReportStatus_EMAIL,
			storage.ReportStatus_SCHEDULED, nil)
		if err != nil {
			log.Errorf("Error submitting scheduled report request for '%s': %s", reportConfig.GetName(), err)
		}
		_, err = s.SubmitReportRequest(scheduledCtx, reportReq, false)
		if err != nil {
			log.Errorf("Error submitting scheduled report request for '%s': %s", reportConfig.GetName(), err)
		}
	}
}

func (s *scheduler) queuePendingReports() {
	pendingReportsQuery := search.NewQueryBuilder().
		AddExactMatches(search.ReportState, storage.ReportStatus_WAITING.String(), storage.ReportStatus_PREPARING.String()).
		WithPagination(search.NewPagination().AddSortOption(search.NewSortOption(search.ReportQueuedTime))).
		ProtoQuery()
	pendingReports, err := s.reportSnapshotStore.SearchReportSnapshots(scheduledCtx, pendingReportsQuery)
	if err != nil {
		log.Errorf("Error finding pending reports: %s", err)
		return
	}

	for _, snap := range pendingReports {
		_, found, err := s.reportConfigDatastore.GetReportConfiguration(scheduledCtx, snap.GetReportConfigurationId())
		if err != nil {
			log.Errorf("Error rescheduling pending report job for report config ID '%s': %s", snap.GetReportConfigurationId(), err)
			continue
		}
		if !found {
			log.Errorf("Report configuration with ID %s had pending reports but the configuration no longer exists",
				snap.GetReportConfigurationId())
			continue
		}

		collection, found, err := s.collectionDatastore.Get(scheduledCtx, snap.GetCollection().GetId())
		if err != nil {
			log.Errorf("Error finding collection ID '%s': %s", snap.GetCollection().GetId(), err)
			continue
		}
		if !found {
			log.Errorf("Collection ID '%s' not found", snap.GetCollection().GetId())
		}

		_, err = s.SubmitReportRequest(scheduledCtx, &reportGen.ReportRequest{
			ReportSnapshot: snap,
			Collection:     collection,
		}, true)
		if err != nil {
			log.Errorf("Error rescheduling pending report job for report config ID '%s': %s", snap.GetReportConfigurationId(), err)
		}
	}
}

func (s *scheduler) queueScheduledReports() {
	query := search.NewQueryBuilder().
		AddExactMatches(search.ReportType, storage.ReportConfiguration_VULNERABILITY.String()).
		ProtoQuery()
	filteredQ := common.WithoutV1ReportConfigs(query)
	reportConfigs, err := s.reportConfigDatastore.GetReportConfigurations(scheduledCtx, filteredQ)
	if err != nil {
		log.Errorf("Error finding scheduled reports: %s", err)
		return
	}
	for _, rc := range reportConfigs {
		if rc.GetSchedule() != nil {
			if err := s.UpsertReportSchedule(rc); err != nil {
				log.Errorf("Error queuing scheduled report for report configuration with ID %s: %v", rc.GetId(), err)
			}
		}
	}
}

/* Utility Functions */

// findAndRemoveFromQueue will find the first element that matches the given predicate and returns ReportRequest from that element
// Elements with values that are not of type *reportGen.ReportRequest will be skipped.
// Note: This function does not lock the queue, so any locks to prevent race conditions must be taken by the caller.
func findAndRemoveFromQueue(reportRequestsQueue *list.List, pred func(req *reportGen.ReportRequest) bool) *reportGen.ReportRequest {
	var toRemove *list.Element
	cur := reportRequestsQueue.Front()
	for cur != nil {
		req, ok := cur.Value.(*reportGen.ReportRequest)
		if ok && pred(req) {
			toRemove = cur
			break
		}
		cur = cur.Next()
	}
	if toRemove == nil {
		return nil
	}
	return reportRequestsQueue.Remove(toRemove).(*reportGen.ReportRequest)
}

// Validate report snapshot and store it to db if validation succeeds.
// Will return report_id if successful.
// Validation will check if the user requesting the report doesn't already have a pending report for the same config
func (s *scheduler) validateAndPersistSnapshot(ctx context.Context, snapshot *storage.ReportSnapshot, reSubmission bool) (string, error) {
	s.dbLock.Lock()
	defer s.dbLock.Unlock()

	if snapshot.GetReportStatus().GetReportRequestType() == storage.ReportStatus_ON_DEMAND {
		userHasAnotherReport, err := s.doesUserHavePendingReport(snapshot.GetReportConfigurationId(), snapshot.GetRequester().GetId())
		if err != nil {
			return "", err
		}
		if userHasAnotherReport {
			return "", errors.Wrapf(errox.AlreadyExists, "User already has a report running for config ID '%s'",
				snapshot.GetReportConfigurationId())
		}
	}

	var err error
	if !reSubmission {
		snapshot.ReportId, err = s.reportSnapshotStore.AddReportSnapshot(ctx, snapshot)
	} else {
		err = s.reportSnapshotStore.UpdateReportSnapshot(ctx, snapshot)
	}

	if err != nil {
		return "", err
	}
	return snapshot.GetReportId(), nil
}

func (s *scheduler) doesUserHavePendingReport(configID string, userID string) (bool, error) {
	query := search.NewQueryBuilder().
		AddExactMatches(search.ReportConfigID, configID).
		AddExactMatches(search.ReportState, storage.ReportStatus_WAITING.String(), storage.ReportStatus_PREPARING.String()).
		AddExactMatches(search.ReportRequestType, storage.ReportStatus_ON_DEMAND.String()).
		ProtoQuery()
	runningReports, err := s.reportSnapshotStore.SearchReportSnapshots(scheduledCtx, query)
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
