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
	"github.com/stackrox/rox/central/graphql/resolvers/loaders"
	reportConfigDS "github.com/stackrox/rox/central/reportconfigurations/datastore"
	reportMetadataDS "github.com/stackrox/rox/central/reports/metadata/datastore"
	reportGen "github.com/stackrox/rox/central/reports/scheduler/v2/reportgenerator"
	collectionDS "github.com/stackrox/rox/central/resourcecollection/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/grpc/authz/allow"
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

	scheduledCtx = resolvers.SetAuthorizerOverride(loaders.WithLoaderContext(sac.WithAllAccess(context.Background())), allow.Anonymous())
)

// Scheduler maintains the schedules for reports
type Scheduler interface {
	UpsertReportSchedule(reportConfig *storage.ReportConfiguration) error
	RemoveReportSchedule(reportConfigID string)
	SubmitReportRequest(request *reportGen.ReportRequest, reSubmission bool) (string, error)
	CancelReportRequest(ctx context.Context, reportID string) error
	Start()
	Stop()
}

type scheduler struct {
	// Used to map reportConfigs to their cron jobs. This is only used for scheduled reports, On-demand reports are directly added to reportsQueue
	reportConfigToEntryIDs map[string]cron.EntryID

	reportConfigDatastore reportConfigDS.DataStore
	reportMetadataStore   reportMetadataDS.DataStore
	collectionDatastore   collectionDS.DataStore
	reportGenerator       reportGen.ReportGenerator

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
func New(reportConfigDatastore reportConfigDS.DataStore, reportMetadataStore reportMetadataDS.DataStore,
	collectionDatastore collectionDS.DataStore, reportGenerator reportGen.ReportGenerator) Scheduler {
	cronScheduler := cron.New()
	cronScheduler.Start()
	ourSchema, err := graphql.ParseSchema(resolvers.Schema(), resolvers.New())
	if err != nil {
		panic(err)
	}
	return newSchedulerImpl(reportConfigDatastore, reportMetadataStore, collectionDatastore, reportGenerator,
		cronScheduler, ourSchema)
}

func newSchedulerImpl(reportConfigDatastore reportConfigDS.DataStore, reportMetadataStore reportMetadataDS.DataStore,
	collectionDatastore collectionDS.DataStore, reportGenerator reportGen.ReportGenerator,
	cronScheduler *cron.Cron, schema *graphql.Schema) *scheduler {
	s := &scheduler{
		reportConfigToEntryIDs: make(map[string]cron.EntryID),
		reportConfigDatastore:  reportConfigDatastore,
		reportMetadataStore:    reportMetadataStore,
		collectionDatastore:    collectionDatastore,
		reportGenerator:        reportGenerator,
		reportRequestsQueue:    list.New(),
		readyForReports:        concurrency.NewSignal(),
		runningReportConfigs:   set.NewStringSet(),
		Schema:                 schema,

		stopper:         concurrency.NewStopper(),
		cron:            cronScheduler,
		concurrencySema: semaphore.NewWeighted(int64(env.ReportExecutionMaxConcurrency.IntegerSetting())),
	}
	return s
}

/* Concurrency and scheduling functions */

// Start scheduler. A scheduler instance can only call Start once. It cannot be re-started once stopped.
// This func will log errors if the scheduler fails to start.
func (s *scheduler) Start() {
	if s.isStopped.Load() {
		log.Error("Scheduler already stopped. It cannot be re-started once stopped.")
		return
	}
	swapped := s.isStarted.CompareAndSwap(false, true)
	if !swapped {
		log.Error("Scheduler already running")
		return
	}
	s.queuePendingReports(scheduledCtx)
	s.queueScheduledReports(scheduledCtx)
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
			log.Infof("Executing report '%s' at %v", reportRequest.ReportConfig.GetName(), time.Now().Format(time.RFC822))
			go s.runSingleReport(reportRequest)
		}
	}
}

func (s *scheduler) selectNextRunnableReport() *reportGen.ReportRequest {
	s.schedulerLock.Lock()
	defer s.schedulerLock.Unlock()

	request := findAndRemoveFromQueue(s.reportRequestsQueue, func(req *reportGen.ReportRequest) bool {
		return !s.runningReportConfigs.Contains(req.ReportConfig.GetId())
	})
	if request == nil {
		return nil
	}
	s.runningReportConfigs.Add(request.ReportConfig.GetId())
	return request
}

func (s *scheduler) runSingleReport(req *reportGen.ReportRequest) {
	defer s.readyForReports.Signal()
	defer s.concurrencySema.Release(1)
	defer s.removeFromRunningReportConfigs(req.ReportConfig.GetId())

	s.reportGenerator.ProcessReportRequest(req)
}

func (s *scheduler) removeFromRunningReportConfigs(configID string) {
	s.schedulerLock.Lock()
	defer s.schedulerLock.Unlock()
	s.runningReportConfigs.Remove(configID)
}

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

// CancelReportRequest cancels a report that is still waiting in queue.
// If the report is already being prepared or has completed execution, it cannot be cancelled.
func (s *scheduler) CancelReportRequest(ctx context.Context, reportID string) error {
	metadata, found, err := s.reportMetadataStore.Get(ctx, reportID)
	if err != nil {
		return errors.Errorf("Error finding report ID '%s': %s", reportID, err)
	}
	if !found {
		return errors.Errorf("Report ID '%s' not found", reportID)
	}
	if metadata.ReportStatus.RunState == storage.ReportStatus_SUCCESS ||
		metadata.ReportStatus.RunState == storage.ReportStatus_FAILURE {
		return nil
	}

	if !s.tryRemoveReportFromQueue(reportID) {
		return errors.Errorf("Cannot cancel. Report ID '%s' is already being prepared", reportID)
	}
	err = s.reportMetadataStore.DeleteReportMetadata(ctx, reportID)
	if err != nil {
		return errors.Errorf("Error deleting report ID '%s' from storage", reportID)
	}

	return nil
}

func (s *scheduler) tryRemoveReportFromQueue(reportID string) bool {
	s.schedulerLock.Lock()
	defer s.schedulerLock.Unlock()

	request := findAndRemoveFromQueue(s.reportRequestsQueue, func(req *reportGen.ReportRequest) bool {
		return req.ReportMetadata.GetReportId() == reportID
	})
	return request != nil
}

// SubmitReportRequest submits a report for scheduling either on demand or on a schedule.
// If it's on demand, it will begin running if there isn't already one running from the same requesting user.
// However, the same report can be run concurrently by different users including the system.
func (s *scheduler) SubmitReportRequest(request *reportGen.ReportRequest, reSubmission bool) (string, error) {
	err := reportGen.ValidateReportRequest(request)
	if err != nil {
		return "", err
	}
	request.Ctx = selectContext(request)

	collection, found, err := s.collectionDatastore.Get(request.Ctx, request.ReportConfig.GetResourceScope().GetCollectionId())
	if err != nil {
		return "", errors.Wrapf(err, "Error finding collection ID '%s'", request.ReportConfig.GetResourceScope().GetCollectionId())
	}
	if !found {
		return "", errors.Errorf("Collection ID '%s' not found", request.ReportConfig.GetResourceScope().GetCollectionId())
	}
	request.Collection = collection

	if request.ReportMetadata == nil || request.ReportMetadata.ReportStatus == nil {
		return "", errors.New("Inva")
	}
	request.ReportMetadata.ReportStatus.RunState = storage.ReportStatus_WAITING
	request.ReportMetadata.ReportStatus.QueuedAt = types.TimestampNow()
	request.ReportMetadata.ReportId, err = s.validateAndPersistMetadata(request.Ctx, request.ReportMetadata, reSubmission)
	if err != nil {
		return "", err
	}

	s.appendToReportsQueue(request)

	return request.ReportMetadata.GetReportId(), nil
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
		_, err := s.SubmitReportRequest(&reportGen.ReportRequest{
			ReportConfig: reportConfig,
			ReportMetadata: &storage.ReportMetadata{
				ReportConfigId: reportConfig.Id,
				ReportStatus: &storage.ReportStatus{
					RunState:                 storage.ReportStatus_WAITING,
					ReportRequestType:        storage.ReportStatus_SCHEDULED,
					ReportNotificationMethod: storage.ReportStatus_EMAIL,
				},
			},
		}, false)
		if err != nil {
			log.Errorf("Error submitting scheduled report request for '%s': %s", reportConfig.GetName(), err)
		}
	}
}

func (s *scheduler) queuePendingReports(ctx context.Context) {
	pendingReportsQuery := search.NewQueryBuilder().
		AddExactMatches(search.ReportState, storage.ReportStatus_WAITING.String(), storage.ReportStatus_PREPARING.String()).
		WithPagination(search.NewPagination().AddSortOption(search.NewSortOption(search.ReportQueuedTime))).
		ProtoQuery()
	pendingReports, err := s.reportMetadataStore.SearchReportMetadatas(ctx, pendingReportsQuery)
	if err != nil {
		log.Errorf("Error finding pending reports: %s", err)
		return
	}

	for _, report := range pendingReports {
		reportConfig, found, err := s.reportConfigDatastore.GetReportConfiguration(ctx, report.GetReportConfigId())
		if err != nil {
			log.Errorf("Error rescheduling pending report for report config ID '%s': %s", report.GetReportConfigId(), err)
			continue
		}
		if !found {
			log.Warnf("Report configuration with ID %s had pending reports but the configuration no longer exists",
				report.GetReportConfigId())
			continue
		}
		_, err = s.SubmitReportRequest(&reportGen.ReportRequest{
			ReportConfig:   reportConfig,
			ReportMetadata: report,
		}, true)
		if err != nil {
			log.Errorf("Error rescheduling pending report for report config '%s': %s", report.GetReportConfigId(), err)
		}
	}
}

func (s *scheduler) queueScheduledReports(ctx context.Context) {
	query := search.NewQueryBuilder().
		AddExactMatches(search.ReportType, storage.ReportConfiguration_VULNERABILITY.String()).
		ProtoQuery()
	reportConfigs, err := s.reportConfigDatastore.GetReportConfigurations(ctx, query)
	if err != nil {
		log.Errorf("Error finding scheduled reports: %s", err)
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

// Validate report metadata of the requested report and store it to db if validation succeeds.
// Will return report_id if successful.
// Validation will check if the user requesting the report doesn't already have a pending report for the same config
func (s *scheduler) validateAndPersistMetadata(ctx context.Context, metadata *storage.ReportMetadata, reSubmission bool) (string, error) {
	s.dbLock.Lock()
	defer s.dbLock.Unlock()

	if metadata.GetReportStatus().GetReportRequestType() == storage.ReportStatus_ON_DEMAND {
		userHasAnotherReport, err := s.doesUserHavePendingReport(ctx, metadata.GetReportConfigId(), metadata.GetRequester().GetId())
		if err != nil {
			return "", err
		}
		if userHasAnotherReport {
			return "", errors.Errorf("User already has a report running for config ID '%s'", metadata.GetReportConfigId())
		}
	}

	var err error
	if !reSubmission {
		metadata.ReportId, err = s.reportMetadataStore.AddReportMetadata(ctx, metadata)
	} else {
		err = s.reportMetadataStore.UpdateReportMetadata(ctx, metadata)
	}

	if err != nil {
		return "", err
	}
	return metadata.ReportId, nil
}

func (s *scheduler) doesUserHavePendingReport(ctx context.Context, configID string, userID string) (bool, error) {
	query := search.NewQueryBuilder().
		AddExactMatches(search.ReportConfigID, configID).
		AddExactMatches(search.ReportState, storage.ReportStatus_WAITING.String(), storage.ReportStatus_PREPARING.String()).
		AddExactMatches(search.ReportRequestType, storage.ReportStatus_ON_DEMAND.String()).
		ProtoQuery()
	runningReports, err := s.reportMetadataStore.SearchReportMetadatas(ctx, query)
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

func selectContext(req *reportGen.ReportRequest) context.Context {
	if req.Ctx == nil {
		return scheduledCtx
	}
	return req.Ctx
}
