package v2

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"text/template"
	"time"

	"github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/reports/common"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/branding"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/mathutil"
	"github.com/stackrox/rox/pkg/notifiers"
	"github.com/stackrox/rox/pkg/retry"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/templates"
	"github.com/stackrox/rox/pkg/timestamp"
)

func (s *scheduler) generateAndSendReport(req *ReportRequest, collection *storage.ResourceCollection) error {
	err := s.upsertReportStatus(req.Ctx, req.ReportMetadata, storage.ReportStatus_PREPARING)
	if err != nil {
		return errors.Wrap(err, "Error changing report status to PREPARING")
	}

	var dataStartTime *types.Timestamp
	if req.ReportConfig.GetVulnReportFilters().GetSinceLastSentScheduledReport() {
		dataStartTime, err = s.lastSuccesfulScheduledReportTime(req.Ctx, req.ReportConfig)
		if err != nil {
			return errors.Wrap(err, "Error finding last successful scheduled report time")
		}
	} else if req.ReportConfig.GetVulnReportFilters().GetSinceStartDate() != nil {
		dataStartTime = req.ReportConfig.GetVulnReportFilters().GetSinceStartDate()
	}

	// Get the results of running the report query
	deployedImgData, watchedImgData, err := s.getReportData(req.Ctx, req.ReportConfig, collection, dataStartTime)
	if err != nil {
		return err
	}

	// Format results into CSV
	zippedCSVData, err := common.Format(deployedImgData, watchedImgData)
	if err != nil {
		return err
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
		return errors.Wrap(err, "error formatting the report email text")
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
		return errorList.ToError()
	}

	req.ReportMetadata.ReportStatus.CompletedAt = types.TimestampNow()
	err = s.upsertReportStatus(req.Ctx, req.ReportMetadata, storage.ReportStatus_SUCCESS)
	if err != nil {
		return errors.Wrap(err, "Error changing report status to SUCCESS")
	}
	return nil
}

func (s *scheduler) getReportData(ctx context.Context, rc *storage.ReportConfiguration, collection *storage.ResourceCollection,
	dataStartTime *types.Timestamp) ([]common.DeployedImagesResult, []common.WatchedImagesResult, error) {
	var deployedImgResults []common.DeployedImagesResult
	var watchedImgResults []common.WatchedImagesResult
	rQuery, err := s.buildReportQuery(ctx, rc, collection, dataStartTime)
	if err != nil {
		return nil, nil, err
	}

	if filterOnImageType(rc.GetVulnReportFilters().GetImageTypes(), storage.VulnerabilityReportFilters_DEPLOYED) {
		deploymentIds, err := s.getDeploymentIDs(ctx, rQuery.DeploymentsQuery)
		if err != nil {
			return nil, nil, err
		}
		result, err := s.runPaginatedDeploymentsQuery(ctx, rQuery.CveFieldsQuery, deploymentIds)
		if err != nil {
			return nil, nil, err
		}
		result.Deployments = orderByClusterAndNamespace(result.Deployments)
		deployedImgResults = append(deployedImgResults, result)
	}

	if filterOnImageType(rc.GetVulnReportFilters().GetImageTypes(), storage.VulnerabilityReportFilters_WATCHED) {
		watchedImages, err := s.getWatchedImages(ctx)
		if err != nil {
			return nil, nil, err
		}
		result, err := s.runPaginatedImagesQuery(ctx, rQuery.CveFieldsQuery, watchedImages)
		if err != nil {
			return nil, nil, err
		}
		watchedImgResults = append(watchedImgResults, result)
	}

	return deployedImgResults, watchedImgResults, nil
}

func (s *scheduler) buildReportQuery(ctx context.Context, rc *storage.ReportConfiguration,
	collection *storage.ResourceCollection, dataStartTime *types.Timestamp) (*common.ReportQuery, error) {
	qb := common.NewVulnReportQueryBuilder(collection, rc.GetVulnReportFilters(), s.collectionQueryResolver,
		timestamp.FromProtobuf(dataStartTime).GoTime())
	rQuery, err := qb.BuildQuery(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "error building report query")
	}
	return rQuery, nil
}

// Returns vuln report data from deployments matched by the collection.
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
			strings.Join(deploymentIds[offset:mathutil.MinInt(offset+paginationLimit, len(deploymentIds))], ","))
		r, err := execQuery[common.DeployedImagesResult](ctx, s, deployedImagesReportQuery, deployedImagesReportQueryOpName,
			scopeQuery, cveQuery, nil)
		if err != nil {
			return r, err
		}
		resultData.Deployments = append(resultData.Deployments, r.Deployments...)
		offset += paginationLimit
	}
	return resultData, nil
}

// Returns vuln report data for watched images
func (s *scheduler) runPaginatedImagesQuery(ctx context.Context, cveQuery string, watchedImages []string) (common.WatchedImagesResult, error) {
	offset := paginatedQueryStartOffset
	var resultData common.WatchedImagesResult
	for {
		if offset >= len(watchedImages) {
			break
		}
		scopeQuery := fmt.Sprintf("%s:%s", search.ImageName.String(),
			strings.Join(watchedImages[offset:mathutil.MinInt(offset+paginationLimit, len(watchedImages))], ","))
		sortOpt := map[string]interface{}{
			"field": search.ImageName.String(),
			"aggregateBy": map[string]interface{}{
				"aggregateFunc": "",
				"distinct":      true,
			},
		}
		r, err := execQuery[common.WatchedImagesResult](ctx, s, watchedImagesReportQuery, watchedImagesReportQueryOpName,
			scopeQuery, cveQuery, sortOpt)
		if err != nil {
			return r, err
		}
		resultData.Images = append(resultData.Images, r.Images...)
		offset += paginationLimit
	}
	return resultData, nil
}

func execQuery[T any](ctx context.Context, sched *scheduler, gqlQuery, opName, scopeQuery, cveQuery string,
	sortOpt map[string]interface{}) (T, error) {
	pagination := map[string]interface{}{
		"offset": paginatedQueryStartOffset,
		"limit":  paginationLimit,
	}
	if sortOpt != nil {
		pagination["sortOptions"] = []interface{}{
			sortOpt,
		}
	}

	response := sched.Schema.Exec(ctx,
		gqlQuery, opName, map[string]interface{}{
			"scopequery": scopeQuery,
			"cvequery":   cveQuery,
			"pagination": pagination,
		})
	if len(response.Errors) > 0 {
		log.Errorf("error running graphql query: %s", response.Errors[0].Message)
		return getZero[T](), response.Errors[0].Err
	}
	var res T
	if err := json.Unmarshal(response.Data, &res); err != nil {
		return getZero[T](), err
	}
	return res, nil
}

/* Utility Functions */

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

func (s *scheduler) getDeploymentIDs(ctx context.Context, deploymentsQuery *v1.Query) ([]string, error) {
	results, err := s.deploymentDatastore.Search(ctx, deploymentsQuery)
	if err != nil {
		return nil, err
	}
	return search.ResultsToIDs(results), nil
}

func (s *scheduler) getWatchedImages(ctx context.Context) ([]string, error) {
	watched, err := s.watchedImageDatastore.GetAllWatchedImages(ctx)
	if err != nil {
		return nil, err
	}
	results := make([]string, 0, len(watched))
	for _, img := range watched {
		results = append(results, img.GetName())
	}
	return results, nil
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

func filterOnImageType(imageTypes []storage.VulnerabilityReportFilters_ImageType,
	target storage.VulnerabilityReportFilters_ImageType) bool {
	for _, typ := range imageTypes {
		if typ == target {
			return true
		}
	}
	return false
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

func getZero[T any]() T {
	var result T
	return result
}
