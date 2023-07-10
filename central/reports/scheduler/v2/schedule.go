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

	reportsQueue           []*ReportRequest
	sendNewReports         chan bool
	runningReportConfigIDs set.StringSet
	Schema                 *graphql.Schema

	/* Concurrency and synchronization related fields */
	stopper concurrency.Stopper

	// Use to synchronize access to reportConfigToEntryIDs map
	cronJobsLock sync.Mutex
	// Use to synchronize access to reportsQueue and runningReportConfigIDs
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
		runningReportConfigIDs:  set.NewStringSet(),
		Schema:                  schema,

		stopper:         concurrency.NewStopper(),
		cron:            cronScheduler,
		concurrencySema: semaphore.NewWeighted(int64(env.ReportExecutionMaxConcurrency.IntegerSetting())),
	}
	return s
}

/* Concurrency and scheduling functions */

func (s *scheduler) Start() {
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

	runnableReportIdx := -1
	for i, reportReq := range s.reportsQueue {
		if !s.runningReportConfigIDs.Contains(reportReq.ReportConfig.GetId()) {
			runnableReportIdx = i
			break
		}
	}
	if runnableReportIdx < 0 {
		// did not find any reports to run
		return nil
	}
	request := s.reportsQueue[runnableReportIdx]
	s.reportsQueue = slices.Delete(s.reportsQueue, runnableReportIdx, runnableReportIdx+1)
	s.runningReportConfigIDs.Add(request.ReportConfig.GetId())
	return request
}

func (s *scheduler) runSingleReport(req *ReportRequest) {
	defer s.notifySchedulerForMoreReports()
	defer s.concurrencySema.Release(1)
	defer s.removeFromRunningReportConfigs(req.ReportConfig.GetId())

	collection, found, err := s.collectionDatastore.Get(req.Ctx, req.ReportConfig.GetResourceScope().GetCollectionId())
	if err != nil {
		s.logErrorAndUpsertReportStatus(
			errors.Wrapf(err, "Error finding collection ID '%s'", req.ReportConfig.GetResourceScope().GetCollectionId()),
			req, "")
		return
	}
	if !found {
		s.logErrorAndUpsertReportStatus(
			errors.Errorf("Collection ID '%s' no longer exists", req.ReportConfig.GetResourceScope().GetCollectionId()),
			req, "")
		return
	}

	err = s.generateAndSendReport(req, collection)
	if err != nil {
		s.logErrorAndUpsertReportStatus(err, req, collection.GetName())
	}
	s.takeReportSnapshot(req, collection.GetName())
}

func (s *scheduler) notifySchedulerForMoreReports() {
	s.sendNewReports <- true
}

func (s *scheduler) removeFromRunningReportConfigs(configID string) {
	s.schedulerLock.Lock()
	defer s.schedulerLock.Unlock()
	s.runningReportConfigIDs.Remove(configID)
}

func (s *scheduler) UpsertReportSchedule(reportConfig *storage.ReportConfiguration) error {
	s.cronJobsLock.Lock()
	defer s.cronJobsLock.Unlock()

	cronSpec := ""
	var err error
	var entryID cron.EntryID
	if reportConfig.GetSchedule() != nil {
		cronSpec, err = schedule.ConvertToCronTab(reportConfig.GetSchedule())
		if err != nil {
			return err
		}
		entryID, err = s.cron.AddFunc(cronSpec, s.reportClosure(reportConfig))
		if err != nil {
			return err
		}
	}

	// Remove the old entry if this is an update
	if oldEntryID, ok := s.reportConfigToEntryIDs[reportConfig.GetId()]; ok {
		s.cron.Remove(oldEntryID)
	}
	// Only add to entries if this is still a scheduled report config
	if cronSpec != "" {
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
		return errors.Errorf("Report ID '%s' does not found", reportID)
	}
	if metadata.ReportStatus.RunState == storage.ReportStatus_SUCCESS ||
		metadata.ReportStatus.RunState == storage.ReportStatus_FAILURE {
		return errors.Errorf("Cannot cancel. Report ID '%s' has already completed execution", reportID)
	}

	if !s.tryRemoveReportFromQueue(reportID) {
		return errors.Errorf("Canot cancel. Report ID '%s' is already being prepared", reportID)
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

	idxToRemove := -1
	for i, req := range s.reportsQueue {
		if req.ReportMetadata.GetReportId() == reportID {
			idxToRemove = i
			break
		}
	}
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

/* Utility Functions */

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

func (s *scheduler) logErrorAndUpsertReportStatus(reportErr error, req *ReportRequest, collectionName string) {
	log.Errorf("Error while running report for config '%s': %s", req.ReportConfig.GetName(), reportErr)

	req.ReportMetadata.ReportStatus.ErrorMsg = reportErr.Error()
	req.ReportMetadata.ReportStatus.CompletedAt = types.TimestampNow()
	err := s.upsertReportStatus(req.Ctx, req.ReportMetadata, storage.ReportStatus_FAILURE)
	if err != nil {
		log.Errorf("Error changing report status to FAILURE for report config '%s', report ID '%s': %s",
			req.ReportConfig.GetName(), req.ReportMetadata.GetReportId(), err)
	}

	s.takeReportSnapshot(req, collectionName)
}

func (s *scheduler) takeReportSnapshot(req *ReportRequest, collectionName string) {
	snapshot := generateReportSnapshot(req.ReportConfig, req.ReportMetadata, collectionName)
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
