package scheduler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"text/template"
	"time"

	"github.com/gogo/protobuf/types"
	"github.com/graph-gophers/graphql-go"
	"github.com/pkg/errors"
	clusterDataStore "github.com/stackrox/rox/central/cluster/datastore"
	"github.com/stackrox/rox/central/graphql/resolvers"
	"github.com/stackrox/rox/central/graphql/resolvers/loaders"
	namespaceDatastore "github.com/stackrox/rox/central/namespace/datastore"
	notifierDataStore "github.com/stackrox/rox/central/notifier/datastore"
	"github.com/stackrox/rox/central/notifier/processor"
	"github.com/stackrox/rox/central/notifiers"
	reportConfigDS "github.com/stackrox/rox/central/reportconfigurations/datastore"
	"github.com/stackrox/rox/central/reports/common"
	roleDataStore "github.com/stackrox/rox/central/role/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/grpc/authz/allow"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/protoconv/schedule"
	"github.com/stackrox/rox/pkg/retry"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/timestamp"
	"gopkg.in/robfig/cron.v2"
)

var (
	log = logging.LoggerForModule()

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

	vulnReportEmailTemplate = `
	Red Hat Advanced Cluster Security for Kubernetes has found vulnerabilities associated with the running container images owned by your organization. Please review the attached vulnerability report {{.WhichVulns}} for {{.DateStr}}.

	To address these findings, please review the impacted software packages in the container images running within deployments you are responsible for and update them to a version containing the fix, if one is available.`

	noVulnsFoundEmailTemplate = `
	Red Hat Advanced Cluster Security for Kubernetes has found zero vulnerabilities associated with the running container images owned by your organization.`

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

	reportConfigDatastore reportConfigDS.DataStore
	notifierDatastore     notifierDataStore.DataStore
	clusterDatastore      clusterDataStore.DataStore
	namespaceDatastore    namespaceDatastore.DataStore
	roleDatastore         roleDataStore.DataStore
	notificationProcessor processor.Processor

	reportsToRun chan *ReportRequest

	stoppedSig concurrency.Signal
	stopped    concurrency.Signal

	Schema *graphql.Schema
}

// ReportRequest is a request to the scheduler to run a scheduled or on demand report
type ReportRequest struct {
	ReportConfig *storage.ReportConfiguration
	OnDemand     bool
	Ctx          context.Context
}

type reportEmailFormat struct {
	WhichVulns string
	DateStr    string
}

// New instantiates a new cron scheduler and supports adding and removing report configurations
func New(reportConfigDS reportConfigDS.DataStore, notifierDS notifierDataStore.DataStore,
	clusterDS clusterDataStore.DataStore, namespaceDS namespaceDatastore.DataStore, roleDS roleDataStore.DataStore,
	notificationProcessor processor.Processor) Scheduler {
	cronScheduler := cron.New()
	cronScheduler.Start()

	ourSchema, err := graphql.ParseSchema(resolvers.Schema(), resolvers.New())
	if err != nil {
		panic(err)
	}
	s := &scheduler{
		reportConfigToEntryIDs: make(map[string]cron.EntryID),
		cron:                   cronScheduler,
		reportConfigDatastore:  reportConfigDS,
		notifierDatastore:      notifierDS,
		clusterDatastore:       clusterDS,
		namespaceDatastore:     namespaceDS,
		roleDatastore:          roleDS,
		notificationProcessor:  notificationProcessor,
		reportsToRun:           make(chan *ReportRequest, 100),
		Schema:                 ourSchema,

		stoppedSig: concurrency.NewSignal(),
		stopped:    concurrency.NewSignal(),
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
	defer s.stopped.Signal()
	for !s.stoppedSig.IsDone() {
		select {
		case <-s.stoppedSig.Done():
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
	namespaces, err := s.namespaceDatastore.GetNamespaces(req.Ctx)
	if err != nil {
		return errors.Wrap(err, "error building report query: unable to get namespaces")
	}

	scope, found, err := s.roleDatastore.GetAccessScope(req.Ctx, rc.GetScopeId())
	if err != nil {
		return errors.Wrap(err, "error building report query: unable to get the resource scope")
	}
	if !found {
		return errors.Errorf("error building report query: resource scope %s not found", scope.GetId())
	}

	qb := common.NewVulnReportQueryBuilder(clusters, namespaces, scope, rc.GetVulnReportFilters(),
		timestamp.FromProtobuf(rc.GetLastSuccessfulRunTime()).GoTime())
	reportQuery, err := qb.BuildQuery()
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
	var messageText string
	if zippedCSVData == nil {
		messageText = noVulnsFoundEmailTemplate
	} else {
		messageText, err = formatMessage(rc)
		if err != nil {
			return errors.Wrap(err, "error formatting the report email text")
		}
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

func formatMessage(rc *storage.ReportConfiguration) (string, error) {
	data := &reportEmailFormat{
		WhichVulns: "for all vulnerabilities",
		DateStr:    time.Now().Format("January 02, 2006"),
	}
	if rc.GetVulnReportFilters().SinceLastReport && rc.GetLastSuccessfulRunTime() != nil {
		data.WhichVulns = fmt.Sprintf("for new vulnerabilities since %s",
			timestamp.FromProtobuf(rc.LastSuccessfulRunTime).GoTime().Format("January 02, 2006"))
	}
	tmpl, err := template.New("emailBody").Parse(vulnReportEmailTemplate)
	if err != nil {
		return "", err
	}
	var tpl bytes.Buffer
	err = tmpl.Execute(&tpl, data)
	if err != nil {
		return "", err
	}
	return tpl.String(), nil
}

func (s *scheduler) getReportData(ctx context.Context, rQuery *common.ReportQuery) ([]common.Result, error) {
	r := make([]common.Result, 0, len(rQuery.ScopeQueries))
	for _, sq := range rQuery.ScopeQueries {
		response := s.Schema.Exec(ctx,
			reportDataQuery, "getVulnReportData", map[string]interface{}{
				"scopequery": sq,
				"cvequery":   rQuery.CveFieldsQuery,
			})
		if len(response.Errors) > 0 {
			return []common.Result{}, response.Errors[0].Err
		}

		var resultData common.Result
		if err := json.Unmarshal(response.Data, &resultData); err != nil {
			return []common.Result{}, err
		}
		r = append(r, resultData)
	}
	return r, nil
}

func (s *scheduler) Start() {
	if !features.VulnReporting.Enabled() {
		return
	}
	go s.runReports()
}

func (s *scheduler) Stop() {
	s.stoppedSig.Signal()
	s.stopped.Wait()
}
