package v2

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync/atomic"
	"text/template"
	"time"

	"github.com/gogo/protobuf/types"
	"github.com/graph-gophers/graphql-go"
	"github.com/pkg/errors"
	deploymentDS "github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/graphql/resolvers"
	"github.com/stackrox/rox/central/graphql/resolvers/loaders"
	notifierDS "github.com/stackrox/rox/central/notifier/datastore"
	reportConfigDS "github.com/stackrox/rox/central/reportconfigurations/datastore"
	"github.com/stackrox/rox/central/reports/common"
	reportMetadataDS "github.com/stackrox/rox/central/reports/metadata/datastore"
	reportSnapshotDS "github.com/stackrox/rox/central/reports/snapshot/datastore"
	collectionDS "github.com/stackrox/rox/central/resourcecollection/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/branding"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/grpc/authz/allow"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/mathutil"
	"github.com/stackrox/rox/pkg/notifier"
	"github.com/stackrox/rox/pkg/notifiers"
	"github.com/stackrox/rox/pkg/protoconv/schedule"
	"github.com/stackrox/rox/pkg/retry"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/templates"
	"github.com/stackrox/rox/pkg/timestamp"
	"gopkg.in/robfig/cron.v2"
)

const (
	deploymentsPaginationLimit = 50

	reportQueryPostgres = `query getVulnReportData($scopequery: String, 
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
						}
	fragment cveFields on ImageVulnerability {
        cve
	    severity
        fixedByVersion
        isFixable
        discoveredAtImage
		link
    }`

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
	CancelReport(reportID string)
	Start()
	Stop()
}

type scheduler struct {
	lock                   sync.Mutex
	cron                   *cron.Cron
	reportConfigToEntryIDs map[string]cron.EntryID

	reportConfigDatastore   reportConfigDS.DataStore
	reportMetadataStore     reportMetadataDS.DataStore
	reportSnapshotStore     reportSnapshotDS.DataStore
	notifierDatastore       notifierDS.DataStore
	deploymentDatastore     deploymentDS.DataStore
	collectionDatastore     collectionDS.DataStore
	collectionQueryResolver collectionDS.QueryResolver
	notificationProcessor   notifier.Processor

	queuedReports          []*ReportRequest
	reportsToRun           chan *ReportRequest
	sendNewReports         chan bool
	numRoutines            atomic.Int32
	runningReportConfigIDs set.StringSet

	stopper concurrency.Stopper

	Schema *graphql.Schema
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
	deploymentDatastore deploymentDS.DataStore, collectionDatastore collectionDS.DataStore,
	collectionQueryRes collectionDS.QueryResolver, notificationProcessor notifier.Processor) Scheduler {
	cronScheduler := cron.New()
	cronScheduler.Start()
	ourSchema, err := graphql.ParseSchema(resolvers.Schema(), resolvers.New())
	if err != nil {
		panic(err)
	}
	return newSchedulerImpl(reportConfigDatastore, reportMetadataStore, reportSnapshotStore, notifierDatastore,
		deploymentDatastore, collectionDatastore, collectionQueryRes, notificationProcessor, cronScheduler, ourSchema)
}

func newSchedulerImpl(reportConfigDatastore reportConfigDS.DataStore, reportMetadataStore reportMetadataDS.DataStore,
	reportSnapshotStore reportSnapshotDS.DataStore, notifierDatastore notifierDS.DataStore,
	deploymentDatastore deploymentDS.DataStore, collectionDatastore collectionDS.DataStore,
	collectionQueryRes collectionDS.QueryResolver, notificationProcessor notifier.Processor,
	cronScheduler *cron.Cron, schema *graphql.Schema) *scheduler {
	s := &scheduler{
		reportConfigToEntryIDs:  make(map[string]cron.EntryID),
		cron:                    cronScheduler,
		reportConfigDatastore:   reportConfigDatastore,
		reportMetadataStore:     reportMetadataStore,
		reportSnapshotStore:     reportSnapshotStore,
		notifierDatastore:       notifierDatastore,
		deploymentDatastore:     deploymentDatastore,
		collectionDatastore:     collectionDatastore,
		collectionQueryResolver: collectionQueryRes,
		notificationProcessor:   notificationProcessor,
		queuedReports:           make([]*ReportRequest, 0),
		reportsToRun:            make(chan *ReportRequest, env.ReportExecutionMaxConcurrency.IntegerSetting()),
		sendNewReports:          make(chan bool, 100),
		runningReportConfigIDs:  set.NewStringSet(),
		Schema:                  schema,
	}
	s.numRoutines.Store(0)
	return s
}

func (s *scheduler) UpsertReportSchedule(reportConfig *storage.ReportConfiguration) error {
	s.lock.Lock()
	defer s.lock.Unlock()

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

func (s *scheduler) RemoveReportSchedule(reportConfigID string) {
	s.lock.Lock()
	defer s.lock.Unlock()

	oldEntryID, exists := s.reportConfigToEntryIDs[reportConfigID]
	if exists {
		s.cron.Remove(oldEntryID)
		delete(s.reportConfigToEntryIDs, reportConfigID)
	}
}

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

	s.lock.Lock()
	s.queuedReports = append(s.queuedReports, request)
	s.lock.Unlock()
	s.sendNewReports <- true

	return request.ReportMetadata.GetReportId(), nil
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

func (s *scheduler) CancelReport(reportID string) {
	//TODO implement me
	panic("implement me")
}

func (s *scheduler) runReports() {
	defer s.stopper.Flow().ReportStopped()
	for {
		select {
		case <-s.stopper.Flow().StopRequested():
			return
		case req := <-s.reportsToRun:
			log.Infof("Executing report '%s' at %v", req.ReportConfig.GetName(), time.Now().Format(time.RFC822))
			go s.sendReportResults(req)
		}
	}
}

func (s *scheduler) scheduleReports() {
	maxConcurrency := int32(env.ReportExecutionMaxConcurrency.IntegerSetting())
	for {
		select {
		case <-s.stopper.Flow().StopRequested():
			return
		case <-s.sendNewReports:
			numRoutines := s.numRoutines.Load()
			// Checking len(s.reportsToRun) < maxConcurrency (which is also the capacity of buffered channel reportsToRun) here
			// will ensure that we never attempt to write to full channel reportsToRun. This will further ensure that
			// writes to reportsToRun in selectRunnableReports() are never blocked.
			if int32(len(s.reportsToRun)) < maxConcurrency && numRoutines < maxConcurrency {
				s.selectRunnableReports(int(maxConcurrency - numRoutines))
			}
		}
	}
}

func (s *scheduler) selectRunnableReports(max int) {
	s.lock.Lock()
	defer s.lock.Unlock()
	indices := set.NewIntSet()
	i := 0
	for indices.Cardinality() < max && i < len(s.queuedReports) {
		if !s.runningReportConfigIDs.Contains(s.queuedReports[i].ReportConfig.GetId()) {
			indices.Add(i)
		}
		i++
	}
	var requests []*ReportRequest
	s.queuedReports, requests = removeSelectedRequests(s.queuedReports, indices)

	selected := make([]*ReportRequest, 0)
	for _, req := range requests {
		metadata := req.ReportMetadata
		metadata.ReportStatus.RunState = storage.ReportStatus_PREPARING
		err := s.reportMetadataStore.UpdateReportMetadata(req.Ctx, metadata)
		if err != nil {
			s.logErrorAndUpsertReportStatus(errors.Wrap(err, "Error updating report status to PREPARING"), req)
		} else {
			selected = append(selected, req)
			s.runningReportConfigIDs.Add(req.ReportConfig.GetId())
		}
	}
	for _, req := range selected {
		// This write should never be blocking. Otherwise it can cause a deadlock.
		s.reportsToRun <- req
	}
}

func (s *scheduler) logErrorAndUpsertReportStatus(err error, req *ReportRequest) {
	log.Errorf("Error running report for config '%s': %s", req.ReportConfig.GetName(), err)
	// TODO : upsert report metadata and report snapshot with error
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

func (s *scheduler) sendReportResults(req *ReportRequest) {
	s.numRoutines.Add(1)
	defer s.notifySchedulerForMoreReports()
	defer s.numRoutines.Add(-1)
	defer s.runningReportConfigIDs.Remove(req.ReportConfig.GetId())

	var dataStartTime *types.Timestamp
	var err error
	if req.ReportConfig.GetVulnReportFilters().GetSinceLastSentScheduledReport() {
		dataStartTime, err = s.lastSuccesfulScheduledReportTime(req.Ctx, req.ReportConfig)
		if err != nil {
			s.logErrorAndUpsertReportStatus(errors.Wrap(err, "Error finding last successful scheduled report time"), req)
			return
		}
	} else if req.ReportConfig.GetVulnReportFilters().GetSinceStartDate() != nil {
		dataStartTime = req.ReportConfig.GetVulnReportFilters().GetSinceStartDate()
	}

	// Get the results of running the report query
	reportData, err := s.getReportData(req.Ctx, req.ReportConfig, dataStartTime)
	if err != nil {
		s.logErrorAndUpsertReportStatus(err, req)
		return
	}

	// Format results into CSV
	zippedCSVData, err := common.Format(reportData)
	if err != nil {
		s.logErrorAndUpsertReportStatus(err, req)
		return
	}

	// If it is an empty report, do not send an attachment in the final notification email and the email body
	// will indicate that no vulns were found
	templateStr := vulnReportEmailTemplate
	if zippedCSVData == nil {
		// If it is an empty report, the email body will indicate that no vulns were found
		templateStr = noVulnsFoundEmailTemplate
	}

	messageText, err := formatMessage(dataStartTime, templateStr)
	if err != nil {
		s.logErrorAndUpsertReportStatus(errors.Wrap(err, "error formatting the report email text"), req)
		return
	}

	errorList := errorhelpers.NewErrorList("Error sending email notifications: ")
	for _, notifierConfig := range req.ReportConfig.GetNotifiers() {
		nf := s.notificationProcessor.GetNotifier(req.Ctx, notifierConfig.GetEmailConfig().GetNotifierId())
		reportNotifier, ok := nf.(notifiers.ReportNotifier)
		if !ok {
			errorList.AddError(errors.Errorf("incorrect type of notifier '%s'", notifierConfig.GetEmailConfig().GetNotifierId()))
			continue
		}
		err := s.retryableSendReportResults(req.Ctx, reportNotifier, notifierConfig.GetEmailConfig().GetMailingLists(),
			zippedCSVData, messageText)
		if err != nil {
			errorList.AddError(errors.Errorf("Error sending email for notifier '%s': %s",
				notifierConfig.GetEmailConfig().GetNotifierId(), err))
		}
	}
	if !errorList.Empty() {
		s.logErrorAndUpsertReportStatus(errorList.ToError(), req)
	}
}

func (s *scheduler) retryableSendReportResults(ctx context.Context, reportNotifier notifiers.ReportNotifier, mailingList []string,
	zippedCSVData *bytes.Buffer, messageText string) error {
	return retry.WithRetry(func() error {
		return reportNotifier.ReportNotify(ctx, zippedCSVData, mailingList, messageText)
	},
		retry.OnlyRetryableErrors(),
		retry.Tries(3),
		retry.BetweenAttempts(func(previousAttempt int) {
			wait := time.Duration(previousAttempt * previousAttempt * 100)
			time.Sleep(wait * time.Millisecond)
		}),
	)
}

func (s *scheduler) notifySchedulerForMoreReports() {
	s.sendNewReports <- true
}

func (s *scheduler) getReportData(ctx context.Context, rc *storage.ReportConfiguration, dataStartTime *types.Timestamp) ([]common.Result, error) {
	var results []common.Result

	if filterOnImageType(rc.GetVulnReportFilters().GetImageTypes(), storage.VulnerabilityReportFilters_DEPLOYED) {
		collection, found, err := s.collectionDatastore.Get(ctx, rc.GetResourceScope().GetCollectionId())
		if err != nil {
			return nil, errors.Wrapf(err, "error building report query: unable to get the collection %s", rc.GetScopeId())
		}
		if !found {
			return nil, errors.Errorf("error building report query: collection with id %s not found", rc.GetScopeId())
		}
		rQuery, err := s.buildDeployedImagesQuery(ctx, rc, collection, dataStartTime)
		if err != nil {
			return nil, err
		}
		deploymentIds, err := s.getDeploymentIDs(ctx, rQuery.DeploymentsQuery)
		if err != nil {
			return nil, err
		}
		result, err := s.runPaginatedDeploymentsQuery(ctx, rQuery.CveFieldsQuery, deploymentIds)
		if err != nil {
			return nil, err
		}
		result.Deployments = orderByClusterAndNamespace(result.Deployments)
		results = append(results, result)
	}

	if filterOnImageType(rc.GetVulnReportFilters().GetImageTypes(), storage.VulnerabilityReportFilters_WATCHED) {
		//TODO implement me
		panic("implement me")
	}

	return results, nil
}

func (s *scheduler) buildDeployedImagesQuery(ctx context.Context, rc *storage.ReportConfiguration,
	collection *storage.ResourceCollection, dataStartTime *types.Timestamp) (*common.ReportQuery, error) {
	qb := common.NewVulnReportQueryBuilder(collection, rc.GetVulnReportFilters(), s.collectionQueryResolver,
		timestamp.FromProtobuf(dataStartTime).GoTime())
	rQuery, err := qb.BuildQuery(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "error building report query")
	}
	return rQuery, nil
}

// Returns vuln report data from deployments matched by embedded resource collection.
func (s *scheduler) runPaginatedDeploymentsQuery(ctx context.Context, cveQuery string, deploymentIds []string) (common.Result, error) {
	offset := paginatedQueryStartOffset
	var resultData common.Result
	for {
		if offset >= len(deploymentIds) {
			break
		}
		// deploymentsQuery is a *v1.Query and graphQL resolvers accept a string queries.
		// With nested collections, deploymentsQuery could be a disjunction of conjunctions like the example below.
		// [(Cluster: c1 AND Namespace: n1 AND Deployment: d1) OR (Cluster: c2 AND Namespace: n2 AND Deployment: d2)]
		// Current string query language doesn't support disjunction of multiple conjunctions, so we cannot build a
		// string equivalent of deploymentsQuery for graphQL. Because of this, we first fetch deploymentIDs from
		// deploymentDatastore using deploymentsQuery and then build string query for graphQL using those deploymentIDs.
		scopeQuery := fmt.Sprintf("%s:%s", search.DeploymentID.String(),
			strings.Join(deploymentIds[offset:mathutil.MinInt(offset+deploymentsPaginationLimit, len(deploymentIds))], ","))
		r, err := s.execReportDataQuery(ctx, reportQueryPostgres, scopeQuery, cveQuery, paginatedQueryStartOffset)
		if err != nil {
			return r, err
		}
		resultData.Deployments = append(resultData.Deployments, r.Deployments...)
		offset += deploymentsPaginationLimit
	}
	return resultData, nil
}

func (s *scheduler) execReportDataQuery(ctx context.Context, gqlQuery, scopeQuery, cveQuery string, offset int) (common.Result, error) {
	response := s.Schema.Exec(ctx,
		gqlQuery, "getVulnReportData", map[string]interface{}{
			"scopequery": scopeQuery,
			"cvequery":   cveQuery,
			"pagination": map[string]interface{}{
				"offset": offset,
				"limit":  deploymentsPaginationLimit,
			},
		})
	if len(response.Errors) > 0 {
		log.Errorf("error running graphql query: %s", response.Errors[0].Message)
		return common.Result{}, response.Errors[0].Err
	}
	var res common.Result
	if err := json.Unmarshal(response.Data, &res); err != nil {
		return common.Result{}, err
	}
	return res, nil
}

func (s *scheduler) getDeploymentIDs(ctx context.Context, deploymentsQuery *v1.Query) ([]string, error) {
	results, err := s.deploymentDatastore.Search(ctx, deploymentsQuery)
	if err != nil {
		return nil, err
	}
	return search.ResultsToIDs(results), nil
}

func (s *scheduler) Start() {
	go s.scheduleReports()
	go s.runReports()
}

func (s *scheduler) Stop() {
	s.stopper.Client().Stop()
	err := s.stopper.Client().Stopped().Wait()
	if err != nil {
		log.Errorf("Error stopping vulnerability report scheduler : %v", err)
	}
}

func filterOnImageType(imageTypes []storage.VulnerabilityReportFilters_ImageType,
	target storage.VulnerabilityReportFilters_ImageType) bool {
	for _, typ := range imageTypes {
		if typ == target {
			return true
		}
	}
	return false
}

func removeSelectedRequests(original []*ReportRequest, indices set.IntSet) ([]*ReportRequest, []*ReportRequest) {
	c := indices.Cardinality()
	removed := make([]*ReportRequest, 0, c)
	remaining := make([]*ReportRequest, 0, len(original)-c)

	for i := 0; i < len(original) && len(removed) < c; i++ {
		if indices.Contains(i) {
			removed = append(removed, original[i])
		} else {
			remaining = append(remaining, original[i])
		}
	}
	return remaining, removed
}

func selectContext(req *ReportRequest) context.Context {
	if req.Ctx == nil {
		return scheduledCtx
	}
	return req.Ctx
}

func formatMessage(dataStartTime *types.Timestamp, emailTemplate string) (string, error) {
	data := &reportEmailFormat{
		BrandedProductName: branding.GetProductName(),
		WhichVulns:         "for all vulnerabilities",
		DateStr:            time.Now().Format("January 02, 2006"),
	}
	if dataStartTime != nil {
		data.WhichVulns = fmt.Sprintf("for new vulnerabilities since %s",
			timestamp.FromProtobuf(dataStartTime).GoTime().Format("January 02, 2006"))
	}
	tmpl, err := template.New("emailBody").Parse(emailTemplate)
	if err != nil {
		return "", err
	}
	return templates.ExecuteToString(tmpl, data)
}

func orderByClusterAndNamespace(deployments []*common.Deployment) []*common.Deployment {
	sort.SliceStable(deployments, func(i, j int) bool {
		if deployments[i].Cluster.GetName() == deployments[j].Cluster.GetName() {
			return deployments[i].Namespace < deployments[j].Namespace
		}
		return deployments[i].Cluster.GetName() < deployments[j].Cluster.GetName()
	})
	return deployments
}
