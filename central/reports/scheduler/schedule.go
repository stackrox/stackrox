package scheduler

import (
	"context"
	"encoding/json"
	"fmt"
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
	"github.com/stackrox/rox/central/notifier/processor"
	"github.com/stackrox/rox/central/notifiers"
	reportConfigDS "github.com/stackrox/rox/central/reportconfigurations/datastore"
	"github.com/stackrox/rox/central/reports/common"
	collectionDataStore "github.com/stackrox/rox/central/resourcecollection/datastore"
	roleDataStore "github.com/stackrox/rox/central/role/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/branding"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/grpc/authz/allow"
	"github.com/stackrox/rox/pkg/logging"
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
	numDeploymentsLimit = 50

	reportDataQuery = `query getVulnReportData($scopequery: String, 
							$cvequery: String, $pagination: Pagination) {
							deployments: deployments(query: $scopequery, pagination: $pagination) {
								cluster {
									name
								}
								namespace
								name
								images {
									name {
										full_name:fullName
									}
									components {
										name
										vulns(query: $cvequery) {
											...cveFields
										}
									}
								}
							}
						}
	fragment cveFields on EmbeddedVulnerability {
        cve
	    severity
        fixedByVersion
        isFixable
        discoveredAtImage
		link
    }`

	reportQueryPostgres = `query getVulnReportData($scopequery: String, 
							$cvequery: String, $pagination: Pagination) {
							deployments: deployments(query: $scopequery, pagination: $pagination) {
								cluster {
									name
								}
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
	notificationProcessor   processor.Processor

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
	collectionQueryRes collectionDataStore.QueryResolver, notificationProcessor processor.Processor) Scheduler {
	cronScheduler := cron.New()
	cronScheduler.Start()

	ourSchema, err := graphql.ParseSchema(resolvers.Schema(), resolvers.New())
	if err != nil {
		panic(err)
	}
	s := &scheduler{
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
		Schema:                  ourSchema,

		stopper: concurrency.NewStopper(),
	}
	return s
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

	clusters, err := s.clusterDatastore.GetClusters(req.Ctx)
	if err != nil {
		return errors.Wrap(err, "error building report query: unable to get clusters")
	}
	namespaces, err := s.namespaceDatastore.GetAllNamespaces(req.Ctx)
	if err != nil {
		return errors.Wrap(err, "error building report query: unable to get namespaces")
	}

	var found bool
	var scope *storage.SimpleAccessScope
	var collection *storage.ResourceCollection
	if !features.ObjectCollections.Enabled() {
		scope, found, err = s.roleDatastore.GetAccessScope(req.Ctx, rc.GetScopeId())
	} else {
		collection, found, err = s.collectionDatastore.Get(req.Ctx, rc.GetScopeId())
	}

	if err != nil {
		return errors.Wrap(err, "error building report query: unable to get the resource scope")
	}
	if !found {
		return errors.Errorf("error building report query: resource scope %s not found", scope.GetId())
	}

	qb := common.NewVulnReportQueryBuilder(clusters, namespaces, scope, collection, rc.GetVulnReportFilters(),
		s.collectionQueryResolver, timestamp.FromProtobuf(rc.GetLastSuccessfulRunTime()).GoTime())
	reportQuery, err := qb.BuildQuery(req.Ctx)
	if err != nil {
		return errors.Wrap(err, "error building report query")
	}

	notifier := s.notificationProcessor.GetNotifier(req.Ctx, rc.GetEmailConfig().GetNotifierId())
	reportNotifier, ok := notifier.(notifiers.ReportNotifier)
	if !ok {
		return errors.Errorf("incorrect notifier type in report config '%s'", rc.GetName())
	}
	// Get the results of running the report query
	reportData, err := s.getReportData(req.Ctx, reportQuery)
	if err != nil {
		return err
	}
	// Format results into CSV
	zippedCSVData, err := common.Format(reportData)
	if err != nil {
		return errors.Wrap(err, "error formatting the report data")
	}
	// If it is an empty report, do not send an attachment in the final notification email and the email body
	// will indicate that no vulns were found

	templateStr := vulnReportEmailTemplate
	if zippedCSVData == nil {
		// If it is an empty report, the email body will indicate that no vulns were found
		templateStr = noVulnsFoundEmailTemplate
	}

	messageText, err := formatMessage(rc, templateStr, time.Now())
	if err != nil {
		return errors.Wrap(err, "error formatting the report email text")
	}

	if err = retry.WithRetry(func() error {
		return reportNotifier.ReportNotify(req.Ctx, zippedCSVData,
			rc.GetEmailConfig().GetMailingLists(), messageText)
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

func (s *scheduler) getReportData(ctx context.Context, rQuery *common.ReportQuery) ([]common.Result, error) {
	if !features.ObjectCollections.Enabled() {
		r := make([]common.Result, 0, len(rQuery.ScopeQueries))
		for _, sq := range rQuery.ScopeQueries {
			resultData, err := s.runPaginatedQuery(ctx, sq, rQuery.CveFieldsQuery, rQuery.DeploymentsQuery)
			if err != nil {
				return nil, err
			}
			r = append(r, resultData)
		}
		return r, nil
	}
	result, err := s.runPaginatedQuery(ctx, "", rQuery.CveFieldsQuery, rQuery.DeploymentsQuery)
	if err != nil {
		return nil, err
	}
	result = groupByClusterAndNamespace(result)
	return []common.Result{result}, nil
}

func (s *scheduler) runPaginatedQuery(ctx context.Context, scopeQuery, cveQuery string, deploymentsQuery *v1.Query) (common.Result, error) {
	offset := 0
	var resultData common.Result
	for {
		var gqlQuery string
		gqlPaginationOffset := offset
		if env.PostgresDatastoreEnabled.BooleanSetting() {
			gqlQuery = reportQueryPostgres
			if features.ObjectCollections.Enabled() {
				deploymentIds, err := s.getDeploymentIDs(ctx, deploymentsQuery, int32(offset))
				if err != nil || len(deploymentIds) == 0 {
					return common.Result{}, err
				}
				scopeQuery = fmt.Sprintf("%s:%q", search.DeploymentID.String(), strings.Join(deploymentIds, ","))
				log.Infof("ROX-12629 : deployments scopeQuery : %s", scopeQuery)
				gqlPaginationOffset = 0
			}
		} else {
			gqlQuery = reportDataQuery
		}
		response := s.Schema.Exec(ctx,
			gqlQuery, "getVulnReportData", map[string]interface{}{
				"scopequery": scopeQuery,
				"cvequery":   cveQuery,
				"pagination": map[string]interface{}{
					"offset": gqlPaginationOffset,
					"limit":  numDeploymentsLimit,
				},
			})
		if len(response.Errors) > 0 {
			log.Errorf("error running graphql query: %s", response.Errors[0].Message)
			return common.Result{}, response.Errors[0].Err
		}
		var r common.Result
		if err := json.Unmarshal(response.Data, &r); err != nil {
			return common.Result{}, err
		}
		resultData.Deployments = append(resultData.Deployments, r.Deployments...)
		if len(r.Deployments) < numDeploymentsLimit {
			break
		}
		offset += len(r.Deployments)
	}
	return resultData, nil
}

func (s *scheduler) getDeploymentIDs(ctx context.Context, deploymentsQuery *v1.Query, offset int32) ([]string, error) {
	deploymentsQuery.Pagination = &v1.QueryPagination{
		Limit:  numDeploymentsLimit,
		Offset: offset,
	}
	results, err := s.deploymentDatastore.SearchDeployments(ctx, deploymentsQuery)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(results))
	for _, res := range results {
		ids = append(ids, res.GetId())
	}
	return ids, nil
}

func groupByClusterAndNamespace(result common.Result) common.Result {
	groupedDeployments := make([]*common.Deployment, 0, len(result.Deployments))
	deploymentsByCluster := make(map[string][]*common.Deployment)
	for _, deployment := range result.Deployments {
		clusterName := deployment.Cluster.GetName()
		deploymentsByCluster[clusterName] = append(deploymentsByCluster[clusterName], deployment)
	}

	for _, deployments := range deploymentsByCluster {
		deploymentsByNamespace := make(map[string][]*common.Deployment)
		for _, deployment := range deployments {
			deploymentsByNamespace[deployment.Namespace] = append(deploymentsByNamespace[deployment.Namespace], deployment)
		}
		for _, deps := range deploymentsByNamespace {
			groupedDeployments = append(groupedDeployments, deps...)
		}
	}

	return common.Result{Deployments: groupedDeployments}
}

func (s *scheduler) Start() {
	go s.runReports()
}

func (s *scheduler) Stop() {
	s.stopper.Client().Stop()
	_ = s.stopper.Client().Stopped().Wait()
}
