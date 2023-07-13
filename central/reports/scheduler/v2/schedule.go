package v2

import (
	"context"
	"time"

	"github.com/gogo/protobuf/types"
	"github.com/graph-gophers/graphql-go"
	"github.com/pkg/errors"
	deploymentDS "github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/graphql/resolvers"
	"github.com/stackrox/rox/central/graphql/resolvers/loaders"
	notifierDS "github.com/stackrox/rox/central/notifier/datastore"
	reportConfigDS "github.com/stackrox/rox/central/reportconfigurations/datastore"
	reportMetadataDS "github.com/stackrox/rox/central/reports/metadata/datastore"
	reportSnapshotDS "github.com/stackrox/rox/central/reports/snapshot/datastore"
	collectionDS "github.com/stackrox/rox/central/resourcecollection/datastore"
	"github.com/stackrox/rox/central/role/resources"
	watchedImageDS "github.com/stackrox/rox/central/watchedimage/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/grpc/authz/allow"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/notifier"
	"github.com/stackrox/rox/pkg/protoconv/schedule"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sliceutils"
	"github.com/stackrox/rox/pkg/sync"
	"golang.org/x/exp/slices"
	"golang.org/x/sync/semaphore"
	"gopkg.in/robfig/cron.v2"
)

const (
	paginationLimit = 50

	cveFieldsFragment = `fragment cveFields on ImageVulnerability {
                             cve
	                         severity
                             fixedByVersion
                             isFixable
                             discoveredAtImage
		                     link
                         }`

	deployedImagesReportQuery = `query getDeployedImagesReportData($scopequery: String, 
                               $cvequery: String, $pagination: Pagination) {
							       deployments: deployments(query: $scopequery, pagination: $pagination) {
                                       clusterName
								       namespace
                                       name
                                       images {
                                           name {
								               full_name:fullName
								           }
                                           imageComponents {
									           name
									           imageVulnerabilities(query: $cvequery) {
										           ...cveFields
									           }
								           }
							           }
						           }
					           }` +
		cveFieldsFragment
	deployedImagesReportQueryOpName = "getDeployedImagesReportData"

	watchedImagesReportQuery = `query getWatchedImagesReportData($scopequery: String, $cvequery: String, $pagination: Pagination) {
                              images: images(query: $scopequery, pagination: $pagination) {
                                  name {
                                      full_name:fullName
                                  }
                                  imageComponents {
                                      name
                                      imageVulnerabilities(query: $cvequery) {
                                          ...cveFields
                                      }
                                  }
                              }
                          }` +
		cveFieldsFragment
	watchedImagesReportQueryOpName = "getWatchedImagesReportData"

	vulnReportEmailTemplate = `
	{{.BrandedProductName}} has found vulnerabilities associated with the running container images owned by your organization. Please review the attached vulnerability report {{.WhichVulns}} for {{.DateStr}}.

	To address these findings, please review the impacted software packages in the container images running within deployments you are responsible for and update them to a version containing the fix, if one is available.`

	noVulnsFoundEmailTemplate = `
	{{.BrandedProductName}} has found zero vulnerabilities associated with the running container images owned by your organization.`

	paginatedQueryStartOffset = 0
)

var (
	log = logging.LoggerForModule()

	scheduledCtx = resolvers.SetAuthorizerOverride(loaders.WithLoaderContext(sac.WithAllAccess(context.Background())), allow.Anonymous())
)

// Scheduler maintains the schedules for reports
type Scheduler interface {
	UpsertReportSchedule(reportConfig *storage.ReportConfiguration) error
	RemoveReportSchedule(reportConfigID string)
	SubmitReport(request *ReportRequest, reSubmission bool) (string, error)
	CancelReport(ctx context.Context, reportID string) error
	Start()
	Stop()
}

type scheduler struct {
	// Used to map reportConfigs to their cron jobs. This is only used for scheduled reports, On-demand reports are directly added to reportsQueue
	reportConfigToEntryIDs map[string]cron.EntryID

	reportConfigDatastore   reportConfigDS.DataStore
	reportMetadataStore     reportMetadataDS.DataStore
	reportSnapshotStore     reportSnapshotDS.DataStore
	notifierDatastore       notifierDS.DataStore
	deploymentDatastore     deploymentDS.DataStore
	watchedImageDatastore   watchedImageDS.DataStore
	collectionDatastore     collectionDS.DataStore
	collectionQueryResolver collectionDS.QueryResolver
	notificationProcessor   notifier.Processor

	reportsQueue []*ReportRequest
	// This channel is used to signal the scheduler to run a new report if routines are available
	sendNewReports chan bool
	// Stores config IDs for which a report is currently running. Used to make sure only one report per config at a time.
	runningReportConfigs set.StringSet
	Schema               *graphql.Schema

	/* Concurrency and synchronization related fields */
	stopper concurrency.Stopper

	// Use to synchronize access to reportConfigToEntryIDs map
	cronJobsLock sync.Mutex
	// Use to synchronize access to reportsQueue and runningReportConfigs
	schedulerLock sync.Mutex
	// Use to lock any database tables if needed to prevent race conditions
	dbLock sync.Mutex
	// NOTE: Lock only one mutex at a time. Do not lock another mutex when one is already held.
	//      If you need to lock another mutex, you must free the one already locked first.

	cron            *cron.Cron
	concurrencySema *semaphore.Weighted
}

// ReportRequest is a request to the scheduler to run a scheduled or on demand report
type ReportRequest struct {
	ReportConfig   *storage.ReportConfiguration
	ReportMetadata *storage.ReportMetadata
	collection     *storage.ResourceCollection
	dataStartTime  *types.Timestamp
	Ctx            context.Context
}

type reportEmailFormat struct {
	BrandedProductName string
	WhichVulns         string
	DateStr            string
}

// New instantiates a new cron scheduler and supports adding and removing report requests
func New(reportConfigDatastore reportConfigDS.DataStore, reportMetadataStore reportMetadataDS.DataStore,
	reportSnapshotStore reportSnapshotDS.DataStore, notifierDatastore notifierDS.DataStore,
	deploymentDatastore deploymentDS.DataStore, watchedImageDatastore watchedImageDS.DataStore,
	collectionDatastore collectionDS.DataStore, collectionQueryRes collectionDS.QueryResolver,
	notificationProcessor notifier.Processor) Scheduler {
	cronScheduler := cron.New()
	cronScheduler.Start()
	ourSchema, err := graphql.ParseSchema(resolvers.Schema(), resolvers.New())
	if err != nil {
		panic(err)
	}
	return newSchedulerImpl(reportConfigDatastore, reportMetadataStore, reportSnapshotStore, notifierDatastore,
		deploymentDatastore, watchedImageDatastore, collectionDatastore, collectionQueryRes, notificationProcessor, cronScheduler, ourSchema)
}

func newSchedulerImpl(reportConfigDatastore reportConfigDS.DataStore, reportMetadataStore reportMetadataDS.DataStore,
	reportSnapshotStore reportSnapshotDS.DataStore, notifierDatastore notifierDS.DataStore,
	deploymentDatastore deploymentDS.DataStore, watchedImageDatastore watchedImageDS.DataStore,
	collectionDatastore collectionDS.DataStore, collectionQueryRes collectionDS.QueryResolver,
	notificationProcessor notifier.Processor, cronScheduler *cron.Cron, schema *graphql.Schema) *scheduler {
	s := &scheduler{
		reportConfigToEntryIDs:  make(map[string]cron.EntryID),
		reportConfigDatastore:   reportConfigDatastore,
		reportMetadataStore:     reportMetadataStore,
		reportSnapshotStore:     reportSnapshotStore,
		notifierDatastore:       notifierDatastore,
		deploymentDatastore:     deploymentDatastore,
		watchedImageDatastore:   watchedImageDatastore,
		collectionDatastore:     collectionDatastore,
		collectionQueryResolver: collectionQueryRes,
		notificationProcessor:   notificationProcessor,
		reportsQueue:            make([]*ReportRequest, 0),
		sendNewReports:          make(chan bool, 200),
		runningReportConfigs:    set.NewStringSet(),
		Schema:                  schema,

		stopper:         concurrency.NewStopper(),
		cron:            cronScheduler,
		concurrencySema: semaphore.NewWeighted(int64(env.ReportExecutionMaxConcurrency.IntegerSetting())),
	}
	return s
}

/* Concurrency and scheduling functions */

func (s *scheduler) Start() {
	ctx := sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.WorkflowAdministration)))

	s.queuePendingReports(ctx)
	s.queueScheduledReports(ctx)
	go s.runReports()
}

func (s *scheduler) Stop() {
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
		case <-s.sendNewReports:
			reportRequest := s.selectNextRunnableReport()
			if reportRequest == nil {
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

func (s *scheduler) selectNextRunnableReport() *ReportRequest {
	s.schedulerLock.Lock()
	defer s.schedulerLock.Unlock()

	reportIdx := sliceutils.FindMatching(s.reportsQueue, func(req *ReportRequest) bool {
		return !s.runningReportConfigs.Contains(req.ReportConfig.GetId())
	})

	if reportIdx < 0 {
		// did not find any reports to run
		return nil
	}
	request := s.reportsQueue[reportIdx]
	s.reportsQueue = slices.Delete(s.reportsQueue, reportIdx, reportIdx+1)
	s.runningReportConfigs.Add(request.ReportConfig.GetId())
	return request
}

func (s *scheduler) runSingleReport(req *ReportRequest) {
	defer s.notifySchedulerForMoreReports()
	defer s.concurrencySema.Release(1)
	defer s.removeFromRunningReportConfigs(req.ReportConfig.GetId())

	var err error
	if req.ReportConfig.GetVulnReportFilters().GetSinceLastSentScheduledReport() {
		req.dataStartTime, err = s.lastSuccesfulScheduledReportTime(req.Ctx, req.ReportConfig)
		if err != nil {
			s.logAndUpsertError(errors.Wrap(err, "Error finding last successful scheduled report time"), req)
			return
		}
	} else if req.ReportConfig.GetVulnReportFilters().GetSinceStartDate() != nil {
		req.dataStartTime = req.ReportConfig.GetVulnReportFilters().GetSinceStartDate()
	}

	// Change report status to PREPARING
	err = s.upsertReportStatus(req.Ctx, req.ReportMetadata, storage.ReportStatus_PREPARING)
	if err != nil {
		s.logAndUpsertError(errors.Wrap(err, "Error changing report status to PREPARING"), req)
		return
	}

	err = s.generateAndSendReportResult(req)
	if err != nil {
		s.logAndUpsertError(err, req)
		return
	}

	// Change report status to SUCCESS
	req.ReportMetadata.ReportStatus.CompletedAt = types.TimestampNow()
	err = s.upsertReportStatus(req.Ctx, req.ReportMetadata, storage.ReportStatus_SUCCESS)
	if err != nil {
		s.logAndUpsertError(errors.Wrap(err, "Error changing report status to SUCCESS"), req)
		return
	}
	s.takeReportSnapshot(req)
}

func (s *scheduler) notifySchedulerForMoreReports() {
	s.sendNewReports <- true
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

// CancelReport cancels a report that is still waiting in queue.
// If the report is already being prepared or has completed execution, it cannot be cancelled.
func (s *scheduler) CancelReport(ctx context.Context, reportID string) error {
	metadata, found, err := s.reportMetadataStore.Get(ctx, reportID)
	if err != nil {
		return errors.Errorf("Error finding report ID '%s': %s", reportID, err)
	}
	if !found {
		return errors.Errorf("Report ID '%s' not found", reportID)
	}
	if metadata.ReportStatus.RunState == storage.ReportStatus_SUCCESS ||
		metadata.ReportStatus.RunState == storage.ReportStatus_FAILURE {
		return errors.Errorf("Cannot cancel. Report ID '%s' has already been executed", reportID)
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

	idxToRemove := sliceutils.FindMatching(s.reportsQueue, func(req *ReportRequest) bool {
		return req.ReportMetadata.GetReportId() == reportID
	})
	if idxToRemove < 0 {
		return false
	}
	s.reportsQueue = slices.Delete(s.reportsQueue, idxToRemove, idxToRemove+1)
	return true
}

// SubmitReport submits a report for scheduling either on demand or on a schedule.
// If it's on demand, it will begin running if there isn't already one running from the same requesting user.
// However, the same report can be run concurrently by different users including the system.
func (s *scheduler) SubmitReport(request *ReportRequest, reSubmission bool) (string, error) {
	request.Ctx = selectContext(request)
	var err error
	if request.ReportMetadata.GetReportStatus().GetReportRequestType() == storage.ReportStatus_ON_DEMAND {
		userHasAnotherReport, err := s.doesUserHavePendingReport(request.Ctx, request.ReportConfig.GetId(), request.ReportMetadata.GetRequester().GetId())
		if err != nil {
			return "", err
		}
		if userHasAnotherReport {
			return "", errors.Errorf("User already has a running report for config ID %s", request.ReportConfig.GetId())
		}
	}

	collection, found, err := s.collectionDatastore.Get(request.Ctx, request.ReportConfig.GetResourceScope().GetCollectionId())
	if err != nil {
		return "", errors.Wrapf(err, "Error finding collection ID '%s'", request.ReportConfig.GetResourceScope().GetCollectionId())
	}
	if !found {
		return "", errors.Errorf("Collection ID '%s' not founf", request.ReportConfig.GetResourceScope().GetCollectionId())
	}

	request.collection = collection
	request.ReportMetadata.ReportStatus.RunState = storage.ReportStatus_WAITING
	request.ReportMetadata.ReportStatus.QueuedAt = types.TimestampNow()
	if !reSubmission {
		request.ReportMetadata.ReportId, err = s.reportMetadataStore.AddReportMetadata(request.Ctx, request.ReportMetadata)
	} else {
		err = s.reportMetadataStore.UpdateReportMetadata(request.Ctx, request.ReportMetadata)
	}

	if err != nil {
		return "", err
	}

	s.appendToReportsQueue(request)
	s.sendNewReports <- true

	return request.ReportMetadata.GetReportId(), nil
}

func (s *scheduler) appendToReportsQueue(req *ReportRequest) {
	s.schedulerLock.Lock()
	defer s.schedulerLock.Unlock()
	s.reportsQueue = append(s.reportsQueue, req)
}

func (s *scheduler) reportClosure(reportConfig *storage.ReportConfiguration) func() {
	return func() {
		log.Infof("Submitting scheduled report request for '%s' at %v", reportConfig.GetName(), time.Now().Format(time.RFC850))
		_, err := s.SubmitReport(&ReportRequest{
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
		_, err = s.SubmitReport(&ReportRequest{
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
		log.Error("Error finding scheduled reports: %s", err)
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

func (s *scheduler) doesUserHavePendingReport(ctx context.Context, configID string, userID string) (bool, error) {
	s.dbLock.Lock()
	defer s.dbLock.Unlock()

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

func (s *scheduler) lastSuccesfulScheduledReportTime(ctx context.Context, config *storage.ReportConfiguration) (*types.Timestamp, error) {
	query := search.NewQueryBuilder().
		AddExactMatches(search.ReportConfigID, config.GetId()).
		AddExactMatches(search.ReportRequestType, storage.ReportStatus_SCHEDULED.String()).
		AddExactMatches(search.ReportState, storage.ReportStatus_SUCCESS.String()).
		WithPagination(search.NewPagination().
			AddSortOption(search.NewSortOption(search.ReportCompletionTime).Reversed(true)).
			Limit(1)).
		ProtoQuery()
	results, err := s.reportMetadataStore.SearchReportMetadatas(ctx, query)
	if err != nil {
		return nil, errors.Wrap(err, "Error finding last successful scheduled report time")
	}
	if len(results) > 1 {
		return nil, errors.Errorf("Received %d records when only one record is expected", len(results))
	}
	if len(results) == 0 {
		return nil, nil
	}
	return results[0].GetReportStatus().GetCompletedAt(), nil
}

func (s *scheduler) upsertReportStatus(ctx context.Context, metadata *storage.ReportMetadata, status storage.ReportStatus_RunState) error {
	metadata.ReportStatus.RunState = status
	return s.reportMetadataStore.UpdateReportMetadata(ctx, metadata)
}

func selectContext(req *ReportRequest) context.Context {
	if req.Ctx == nil {
		return scheduledCtx
	}
	return req.Ctx
}

func (s *scheduler) logAndUpsertError(reportErr error, req *ReportRequest) {
	log.Errorf("Error while running report for config '%s': %s", req.ReportConfig.GetName(), reportErr)

	req.ReportMetadata.ReportStatus.ErrorMsg = reportErr.Error()
	req.ReportMetadata.ReportStatus.CompletedAt = types.TimestampNow()
	err := s.upsertReportStatus(req.Ctx, req.ReportMetadata, storage.ReportStatus_FAILURE)
	if err != nil {
		log.Errorf("Error changing report status to FAILURE for report config '%s', report ID '%s': %s",
			req.ReportConfig.GetName(), req.ReportMetadata.GetReportId(), err)
	}

	s.takeReportSnapshot(req)
}

func (s *scheduler) takeReportSnapshot(req *ReportRequest) {
	snapshot := generateReportSnapshot(req.ReportConfig, req.ReportMetadata, req.collection.GetName())
	err := s.reportSnapshotStore.AddReportSnapshot(req.Ctx, snapshot)
	if err != nil {
		log.Errorf("Error storing snapshot for report config '%s', report ID '%s': %s",
			req.ReportConfig.GetName(), req.ReportMetadata.GetReportId(), err)
	}
}

func generateReportSnapshot(config *storage.ReportConfiguration, metadata *storage.ReportMetadata, collectionName string) *storage.ReportSnapshot {
	snapshot := &storage.ReportSnapshot{
		ReportId:              metadata.GetReportId(),
		ReportConfigurationId: config.GetId(),
		Name:                  config.GetName(),
		Description:           config.GetDescription(),
		Type:                  storage.ReportSnapshot_VULNERABILITY,
		Filter: &storage.ReportSnapshot_VulnReportFilters{
			VulnReportFilters: config.GetVulnReportFilters(),
		},
		Collection: &storage.CollectionSnapshot{
			Id:   config.GetResourceScope().GetCollectionId(),
			Name: collectionName,
		},
		Schedule:     config.GetSchedule(),
		ReportStatus: metadata.GetReportStatus(),
		Requester:    metadata.GetRequester(),
	}

	notifierSnaps := make([]*storage.NotifierSnapshot, 0, len(config.GetNotifiers()))
	for _, notifierConf := range config.GetNotifiers() {
		notifierSnaps = append(notifierSnaps, &storage.NotifierSnapshot{
			NotifierConfig: &storage.NotifierSnapshot_EmailConfig{
				EmailConfig: &storage.EmailNotifierSnapshot{
					MailingLists: notifierConf.GetEmailConfig().GetMailingLists(),
				},
			},
		})
	}
	snapshot.Notifiers = notifierSnaps
	return snapshot
}
