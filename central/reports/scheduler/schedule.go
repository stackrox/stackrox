package scheduler

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"text/template"
	"time"

	"github.com/gogo/protobuf/types"
	"github.com/graph-gophers/graphql-go"
	"github.com/pkg/errors"
	clusterDataStore "github.com/stackrox/rox/central/cluster/datastore"
	deploymentDataStore "github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/graphql/resolvers"
	"github.com/stackrox/rox/central/graphql/resolvers/loaders"
	namespaceDataStore "github.com/stackrox/rox/central/namespace/datastore"
	notifierDataStore "github.com/stackrox/rox/central/notifier/datastore"
	"github.com/stackrox/rox/central/reports/common"
	reportConfigDS "github.com/stackrox/rox/central/reports/config/datastore"
	collectionDataStore "github.com/stackrox/rox/central/resourcecollection/datastore"
	roleDataStore "github.com/stackrox/rox/central/role/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/branding"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/grpc/authz/allow"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/mathutil"
	"github.com/stackrox/rox/pkg/notifier"
	"github.com/stackrox/rox/pkg/notifiers"
	"github.com/stackrox/rox/pkg/protoconv/schedule"
	"github.com/stackrox/rox/pkg/retry"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
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
	SubmitReport(request *ReportRequest)
	Start()
	Stop()
}

type scheduler struct {
	lock                   sync.Mutex
	cron                   *cron.Cron
	reportConfigToEntryIDs map[string]cron.EntryID

	reportConfigDatastore   reportConfigDS.DataStore
	notifierDatastore       notifierDataStore.DataStore
	clusterDatastore        clusterDataStore.DataStore
	namespaceDatastore      namespaceDataStore.DataStore
	deploymentDatastore     deploymentDataStore.DataStore
	roleDatastore           roleDataStore.DataStore
	collectionDatastore     collectionDataStore.DataStore
	collectionQueryResolver collectionDataStore.QueryResolver
	notificationProcessor   notifier.Processor

	reportsToRun chan *ReportRequest

	stopper concurrency.Stopper

	Schema *graphql.Schema
}

// ReportRequest is a request to the scheduler to run a scheduled or on demand report
type ReportRequest struct {
	ReportConfig *storage.ReportConfiguration
	OnDemand     bool
	Ctx          context.Context
}

type reportEmailFormat struct {
	BrandedProductName string
	WhichVulns         string
	DateStr            string
}

// New instantiates a new cron scheduler and supports adding and removing report configurations
func New(reportConfigDS reportConfigDS.DataStore, notifierDS notifierDataStore.DataStore,
	clusterDS clusterDataStore.DataStore, namespaceDS namespaceDataStore.DataStore,
	deploymentDS deploymentDataStore.DataStore, collectionDS collectionDataStore.DataStore, roleDS roleDataStore.DataStore,
	collectionQueryRes collectionDataStore.QueryResolver, notificationProcessor notifier.Processor) Scheduler {
	cronScheduler := cron.New()
	cronScheduler.Start()

	ourSchema, err := graphql.ParseSchema(resolvers.Schema(), resolvers.New())
	if err != nil {
		panic(err)
	}

	return newSchedulerImpl(reportConfigDS, notifierDS, clusterDS, namespaceDS, deploymentDS, collectionDS, roleDS,
		collectionQueryRes, notificationProcessor, cronScheduler, ourSchema)
}

func newSchedulerImpl(reportConfigDS reportConfigDS.DataStore, notifierDS notifierDataStore.DataStore,
	clusterDS clusterDataStore.DataStore, namespaceDS namespaceDataStore.DataStore,
	deploymentDS deploymentDataStore.DataStore, collectionDS collectionDataStore.DataStore, roleDS roleDataStore.DataStore,
	collectionQueryRes collectionDataStore.QueryResolver, notificationProcessor notifier.Processor,
	cronScheduler *cron.Cron, schema *graphql.Schema) *scheduler {
	return &scheduler{
		reportConfigToEntryIDs:  make(map[string]cron.EntryID),
		cron:                    cronScheduler,
		reportConfigDatastore:   reportConfigDS,
		notifierDatastore:       notifierDS,
		clusterDatastore:        clusterDS,
		namespaceDatastore:      namespaceDS,
		deploymentDatastore:     deploymentDS,
		collectionDatastore:     collectionDS,
		roleDatastore:           roleDS,
		collectionQueryResolver: collectionQueryRes,
		notificationProcessor:   notificationProcessor,
		reportsToRun:            make(chan *ReportRequest, 100),
		Schema:                  schema,

		stopper: concurrency.NewStopper(),
	}
}

func (s *scheduler) reportClosure(reportConfig *storage.ReportConfiguration) func() {
	return func() {
		log.Infof("Submitting report request for '%s' at %v", reportConfig.GetName(), time.Now().Format(time.RFC850))
		s.SubmitReport(&ReportRequest{
			ReportConfig: reportConfig,
			OnDemand:     false,
			Ctx:          scheduledCtx,
		})
	}
}

func (s *scheduler) UpsertReportSchedule(reportConfig *storage.ReportConfiguration) error {
	s.lock.Lock()
	defer s.lock.Unlock()
	cronSpec, err := schedule.ConvertToCronTab(reportConfig.GetSchedule())
	if err != nil {
		return err
	}
	entryID, err := s.cron.AddFunc(cronSpec, s.reportClosure(reportConfig))
	if err != nil {
		return err
	}

	// Remove the old entry if this is an update
	if oldEntryID, ok := s.reportConfigToEntryIDs[reportConfig.GetId()]; ok {
		s.cron.Remove(oldEntryID)
	}
	s.reportConfigToEntryIDs[reportConfig.GetId()] = entryID
	return nil
}

func (s *scheduler) RemoveReportSchedule(id string) {
	s.lock.Lock()
	defer s.lock.Unlock()

	oldEntryID := s.reportConfigToEntryIDs[id]
	s.cron.Remove(oldEntryID)
	delete(s.reportConfigToEntryIDs, id)
}

func (s *scheduler) SubmitReport(reportRequest *ReportRequest) {
	log.Infof("Submitting report '%s' at %v for execution", reportRequest.ReportConfig.GetName(), time.Now().Format(time.RFC822))
	s.reportsToRun <- reportRequest
}

func (s *scheduler) runReports() {
	defer s.stopper.Flow().ReportStopped()
	for {
		select {
		case <-s.stopper.Flow().StopRequested():
			return
		case req := <-s.reportsToRun:
			log.Infof("Executing report '%s' at %v", req.ReportConfig.GetName(), time.Now().Format(time.RFC822))
			err := s.sendReportResults(req)
			if err != nil {
				log.Errorf("error executing report %s: %s", req.ReportConfig.GetName(), err)
			}
			if err := s.updateLastRunStatus(req, err); err != nil {
				log.Errorf("error updating run status for report config '%s': %s", req.ReportConfig.GetName(), err)
			}
		}
	}
}

func (s *scheduler) updateLastRunStatus(req *ReportRequest, err error) error {
	if req.OnDemand {
		return nil
	}
	if err != nil {
		// TODO: @khushboo for more accuracy, save timestamp when the vuln data is pulled aka the query is run
		req.ReportConfig.LastRunStatus = &storage.ReportLastRunStatus{
			ReportStatus: storage.ReportLastRunStatus_FAILURE,
			LastRunTime:  timestamp.Now().GogoProtobuf(),
			ErrorMsg:     err.Error(),
		}
	} else {
		req.ReportConfig.LastRunStatus = &storage.ReportLastRunStatus{
			ReportStatus: storage.ReportLastRunStatus_SUCCESS,
			LastRunTime:  timestamp.Now().GogoProtobuf(),
			ErrorMsg:     "",
		}
		req.ReportConfig.LastSuccessfulRunTime = types.TimestampNow()
	}
	if err = s.UpsertReportSchedule(req.ReportConfig); err != nil {
		return err
	}
	return s.reportConfigDatastore.UpdateReportConfiguration(req.Ctx, req.ReportConfig)
}

func (s *scheduler) sendReportResults(req *ReportRequest) error {
	rc := req.ReportConfig

	notifier := s.notificationProcessor.GetNotifier(req.Ctx, rc.GetEmailConfig().GetNotifierId())
	reportNotifier, ok := notifier.(notifiers.ReportNotifier)
	if !ok {
		return errors.Errorf("incorrect notifier type in report config '%s'", rc.GetName())
	}
	// Get the results of running the report query
	reportData, err := s.getReportData(req.Ctx, rc)
	if err != nil {
		return err
	}
	// Format results into CSV
	zippedCSVData, empty, err := common.Format(reportData)
	if err != nil {
		return errors.Wrap(err, "error formatting the report data")
	}

	// If it is an empty report, do not send an attachment in the final notification email and the email body
	// will indicate that no vulns were found
	templateStr := vulnReportEmailTemplate
	if empty {
		// If it is an empty report, the email body will indicate that no vulns were found
		zippedCSVData = nil
		templateStr = noVulnsFoundEmailTemplate
	}

	messageText, err := formatMessage(rc, templateStr, time.Now())
	if err != nil {
		return errors.Wrap(err, "error formatting the report email text")
	}

	if err = retry.WithRetry(func() error {
		return reportNotifier.ReportNotify(req.Ctx, zippedCSVData,
			rc.GetEmailConfig().GetMailingLists(), "", messageText)
	},
		retry.OnlyRetryableErrors(),
		retry.Tries(3),
		retry.BetweenAttempts(func(previousAttempt int) {
			wait := time.Duration(previousAttempt * previousAttempt * 100)
			time.Sleep(wait * time.Millisecond)
		}),
	); err != nil {
		return err
	}
	log.Infof("Report generation for '%s' completed, email notification sent.", rc.Name)
	return nil
}

func formatMessage(rc *storage.ReportConfiguration, emailTemplate string, date time.Time) (string, error) {
	data := &reportEmailFormat{
		BrandedProductName: branding.GetProductName(),
		WhichVulns:         "for all vulnerabilities",
		DateStr:            date.Format("January 02, 2006"),
	}
	if rc.GetVulnReportFilters().SinceLastReport && rc.GetLastSuccessfulRunTime() != nil {
		data.WhichVulns = fmt.Sprintf("for new vulnerabilities since %s",
			timestamp.FromProtobuf(rc.LastSuccessfulRunTime).GoTime().Format("January 02, 2006"))
	}
	tmpl, err := template.New("emailBody").Parse(emailTemplate)
	if err != nil {
		return "", err
	}
	return templates.ExecuteToString(tmpl, data)
}

func (s *scheduler) getReportData(ctx context.Context, rc *storage.ReportConfiguration) ([]common.DeployedImagesResult, error) {
	collection, found, err := s.collectionDatastore.Get(ctx, rc.GetScopeId())
	if err != nil {
		return nil, errors.Wrapf(err, "error building report query: unable to get the collection %s", rc.GetScopeId())
	}
	if !found {
		return nil, errors.Errorf("error building report query: collection with id %s not found", rc.GetScopeId())
	}
	rQuery, err := s.buildReportQuery(ctx, rc, collection)
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
	return []common.DeployedImagesResult{result}, nil
}

func (s *scheduler) buildReportQuery(ctx context.Context, rc *storage.ReportConfiguration,
	collection *storage.ResourceCollection) (*common.ReportQuery, error) {
	qb := common.NewVulnReportQueryBuilder(collection, rc.GetVulnReportFilters(), s.collectionQueryResolver,
		timestamp.FromProtobuf(rc.GetLastSuccessfulRunTime()).GoTime())
	rQuery, err := qb.BuildQuery(ctx, nil, nil)
	if err != nil {
		return nil, errors.Wrap(err, "error building report query")
	}
	return rQuery, nil
}

// Returns vuln report data from deployments matched by embedded resource collection.
func (s *scheduler) runPaginatedDeploymentsQuery(ctx context.Context, cveQuery string, deploymentIds []string) (common.DeployedImagesResult, error) {
	offset := paginatedQueryStartOffset
	var resultData common.DeployedImagesResult
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

func (s *scheduler) execReportDataQuery(ctx context.Context, gqlQuery, scopeQuery, cveQuery string, offset int) (common.DeployedImagesResult, error) {
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
		return common.DeployedImagesResult{}, response.Errors[0].Err
	}
	var res common.DeployedImagesResult
	if err := json.Unmarshal(response.Data, &res); err != nil {
		return common.DeployedImagesResult{}, err
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
	go s.runReports()
}

func (s *scheduler) Stop() {
	s.stopper.Client().Stop()
	err := s.stopper.Client().Stopped().Wait()
	if err != nil {
		log.Errorf("Error stopping vulnerability report scheduler : %v", err)
	}
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
