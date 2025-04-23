package reportgenerator

import (
	"bytes"
	"context"
	"time"

	"github.com/graph-gophers/graphql-go"
	"github.com/pkg/errors"
	blobDS "github.com/stackrox/rox/central/blob/datastore"
	clusterDS "github.com/stackrox/rox/central/cluster/datastore"
	imageCVEDS "github.com/stackrox/rox/central/cve/image/datastore"
	imageCVE2DS "github.com/stackrox/rox/central/cve/image/v2/datastore"
	deploymentDS "github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/graphql/resolvers"
	"github.com/stackrox/rox/central/graphql/resolvers/loaders"
	namespaceDS "github.com/stackrox/rox/central/namespace/datastore"
	"github.com/stackrox/rox/central/reports/common"
	reportSnapshotDS "github.com/stackrox/rox/central/reports/snapshot/datastore"
	collectionDS "github.com/stackrox/rox/central/resourcecollection/datastore"
	watchedImageDS "github.com/stackrox/rox/central/watchedimage/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/grpc/authz/allow"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/notifier"
	"github.com/stackrox/rox/pkg/notifiers"
	"github.com/stackrox/rox/pkg/postgres"
	pkgSchema "github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/postgres/walker"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/retry"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	pgSearch "github.com/stackrox/rox/pkg/search/postgres"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	log = logging.LoggerForModule()

	reportGenCtx = resolvers.SetAuthorizerOverride(loaders.WithLoaderContext(sac.WithAllAccess(context.Background())), allow.Anonymous())

	deployedImagesQueryParts = &ReportQueryParts{
		Schema: selectSchema(),
		Selects: []*v1.QuerySelect{
			search.NewQuerySelect(search.ImageName).Proto(),
			search.NewQuerySelect(search.Component).Proto(),
			search.NewQuerySelect(search.CVEID).Proto(),
			search.NewQuerySelect(search.CVE).Proto(),
			search.NewQuerySelect(search.Fixable).Proto(),
			search.NewQuerySelect(search.FixedBy).Proto(),
			search.NewQuerySelect(search.Severity).Proto(),
			search.NewQuerySelect(search.CVSS).Proto(),
			search.NewQuerySelect(search.NVDCVSS).Proto(),
			search.NewQuerySelect(search.FirstImageOccurrenceTimestamp).Proto(),
			search.NewQuerySelect(search.Cluster).Proto(),
			search.NewQuerySelect(search.Namespace).Proto(),
			search.NewQuerySelect(search.DeploymentName).Proto(),
			search.NewQuerySelect(search.EPSSProbablity).Proto(),
		},
		Pagination: search.NewPagination().
			AddSortOption(search.NewSortOption(search.Cluster)).
			AddSortOption(search.NewSortOption(search.Namespace)).Proto(),
	}

	watchedImagesQueryParts = &ReportQueryParts{
		Schema: selectSchema(),
		Selects: []*v1.QuerySelect{
			search.NewQuerySelect(search.ImageName).Proto(),
			search.NewQuerySelect(search.Component).Proto(),
			search.NewQuerySelect(search.CVEID).Proto(),
			search.NewQuerySelect(search.CVE).Proto(),
			search.NewQuerySelect(search.Fixable).Proto(),
			search.NewQuerySelect(search.FixedBy).Proto(),
			search.NewQuerySelect(search.Severity).Proto(),
			search.NewQuerySelect(search.CVSS).Proto(),
			search.NewQuerySelect(search.NVDCVSS).Proto(),
			search.NewQuerySelect(search.FirstImageOccurrenceTimestamp).Proto(),
			search.NewQuerySelect(search.EPSSProbablity).Proto(),
		},
		Pagination: search.NewPagination().
			AddSortOption(search.NewSortOption(search.ImageName)).Proto(),
	}
)

type reportGeneratorImpl struct {
	reportSnapshotStore     reportSnapshotDS.DataStore
	deploymentDatastore     deploymentDS.DataStore
	watchedImageDatastore   watchedImageDS.DataStore
	collectionQueryResolver collectionDS.QueryResolver
	notificationProcessor   notifier.Processor
	blobStore               blobDS.Datastore
	clusterDatastore        clusterDS.DataStore
	namespaceDatastore      namespaceDS.DataStore
	imageCVEDatastore       imageCVEDS.DataStore
	imageCVE2Datastore      imageCVE2DS.DataStore
	db                      postgres.DB

	Schema *graphql.Schema
}

type ImageCVEInterface interface {
	GetId() string
	GetCveBaseInfo() *storage.CVEInfo
}

func (rg *reportGeneratorImpl) ProcessReportRequest(req *ReportRequest) {
	// First do some basic validation checks on the request.
	err := ValidateReportRequest(req)
	if err != nil {
		rg.logAndUpsertError(errors.Wrap(err, "Invalid report request"), req)
		return
	}

	if req.ReportSnapshot.GetVulnReportFilters().GetSinceLastSentScheduledReport() {
		req.DataStartTime, err = rg.lastSuccessfulScheduledReportTime(req.ReportSnapshot)
		if err != nil {
			rg.logAndUpsertError(errors.Wrap(err, "Error finding last successful scheduled report time"), req)
			return
		}
	} else if req.ReportSnapshot.GetVulnReportFilters().GetSinceStartDate() != nil {
		sinceStartDate := req.ReportSnapshot.GetVulnReportFilters().GetSinceStartDate()
		req.DataStartTime, err = protocompat.ConvertTimestampToTimeOrError(sinceStartDate)
		if err != nil {
			rg.logAndUpsertError(errors.Wrap(err, "Error finding last successful scheduled report time"), req)
			return
		}
	}

	// Change report status to PREPARING
	err = rg.updateReportStatus(req.ReportSnapshot, storage.ReportStatus_PREPARING)
	if err != nil {
		rg.logAndUpsertError(errors.Wrap(err, "Error changing report status to PREPARING"), req)
		return
	}

	err = rg.generateReportAndNotify(req)
	if err != nil {
		rg.logAndUpsertError(err, req)
		return
	}

	if req.ReportSnapshot.GetReportStatus().GetReportNotificationMethod() == storage.ReportStatus_EMAIL {
		err = rg.updateReportStatus(req.ReportSnapshot, storage.ReportStatus_DELIVERED)
		if err != nil {
			rg.logAndUpsertError(errors.Wrap(err, "Error changing report status to DELIVERED"), req)
		}
	}
}

/* Report generation helper functions */
func (rg *reportGeneratorImpl) generateReportAndNotify(req *ReportRequest) error {
	// Get the results of running the report query
	reportData, err := rg.getReportDataSQF(req.ReportSnapshot, req.Collection, req.DataStartTime)
	if err != nil {
		return err
	}

	// Format results into CSV
	zippedCSVData, err := GenerateCSV(reportData.CVEResponses, req.ReportSnapshot.Name, req.ReportSnapshot.GetVulnReportFilters())
	if err != nil {
		return err
	}

	req.ReportSnapshot.ReportStatus.CompletedAt = protocompat.TimestampNow()
	err = rg.updateReportStatus(req.ReportSnapshot, storage.ReportStatus_GENERATED)
	if err != nil {
		return errors.Wrap(err, "Error changing report status to GENERATED")
	}

	switch req.ReportSnapshot.ReportStatus.ReportNotificationMethod {
	case storage.ReportStatus_DOWNLOAD:
		if err = rg.saveReportData(req.ReportSnapshot.GetReportConfigurationId(),
			req.ReportSnapshot.GetReportId(), zippedCSVData); err != nil {
			return errors.Wrap(err, "error persisting blob")
		}

	case storage.ReportStatus_EMAIL:
		defaultEmailSubject, err := formatEmailSubject(defaultEmailSubjectTemplate, req.ReportSnapshot)
		if err != nil {
			return errors.Wrap(err, "Error generating email subject")
		}
		// If it is an empty report, do not send an attachment in the final notification email and the email body
		// will indicate that no vulns were found
		templateStr := defaultEmailBodyTemplate
		if reportData.NumDeployedImageResults == 0 && reportData.NumWatchedImageResults == 0 {
			// If it is an empty report, the email body will indicate that no vulns were found
			zippedCSVData = nil
			templateStr = defaultNoVulnsEmailBodyTemplate
		}

		defaultEmailBody, err := formatEmailBody(templateStr)
		if err != nil {
			return errors.Wrap(err, "Error generating email body")
		}

		configDetailsHTML, err := formatReportConfigDetails(req.ReportSnapshot, reportData.NumDeployedImageResults,
			reportData.NumWatchedImageResults)
		if err != nil {
			return errors.Wrap(err, "Error adding report config details")
		}

		errorList := errorhelpers.NewErrorList("Error sending email notifications: ")
		for _, notifierSnap := range req.ReportSnapshot.GetNotifiers() {
			nf := rg.notificationProcessor.GetNotifier(reportGenCtx, notifierSnap.GetEmailConfig().GetNotifierId())
			reportNotifier, ok := nf.(notifiers.ReportNotifier)
			if !ok {
				errorList.AddError(errors.Errorf("incorrect type of notifier '%s'", notifierSnap.GetEmailConfig().GetNotifierId()))
				continue
			}
			customBody := notifierSnap.GetEmailConfig().GetCustomBody()
			emailBody := defaultEmailBody
			if customBody != "" {
				emailBody = customBody
			}
			customSubject := notifierSnap.GetEmailConfig().GetCustomSubject()
			emailSubject := defaultEmailSubject
			if customSubject != "" {
				emailSubject = customSubject
			}
			emailBodyWithConfigDetails := addReportConfigDetails(emailBody, configDetailsHTML)
			reportName := req.ReportSnapshot.Name
			err := rg.retryableSendReportResults(reportNotifier, notifierSnap.GetEmailConfig().GetMailingLists(),
				zippedCSVData, emailSubject, emailBodyWithConfigDetails, reportName)
			if err != nil {
				errorList.AddError(errors.Errorf("Error sending email for notifier '%s': %s",
					notifierSnap.GetEmailConfig().GetNotifierId(), err))
			}
		}
		if !errorList.Empty() {
			return errorList.ToError()
		}
	}
	return nil
}

func (rg *reportGeneratorImpl) saveReportData(configID, reportID string, data *bytes.Buffer) error {
	if data == nil {
		return errors.Errorf("No data found for report config %q and id %q", configID, reportID)
	}

	// Store downloadable report in blob storage
	b := &storage.Blob{
		Name:         common.GetReportBlobPath(configID, reportID),
		LastUpdated:  protocompat.TimestampNow(),
		ModifiedTime: protocompat.TimestampNow(),
		Length:       int64(data.Len()),
	}
	return rg.blobStore.Upsert(reportGenCtx, b, data)
}

func (rg *reportGeneratorImpl) getReportDataSQF(snap *storage.ReportSnapshot, collection *storage.ResourceCollection,
	dataStartTime time.Time) (*ReportData, error) {
	rQuery, err := rg.buildReportQuery(snap, collection, dataStartTime)
	if err != nil {
		return nil, err
	}

	cveFilterQuery, err := search.ParseQuery(rQuery.CveFieldsQuery, search.MatchAllIfEmpty())
	if err != nil {
		return nil, err
	}

	numDeployedImageResults := 0
	var cveResponses []*ImageCVEQueryResponse
	if filterOnImageType(snap.GetVulnReportFilters().GetImageTypes(), storage.VulnerabilityReportFilters_DEPLOYED) {
		query := search.ConjunctionQuery(rQuery.DeploymentsQuery, cveFilterQuery)
		query.Pagination = deployedImagesQueryParts.Pagination
		query.Selects = deployedImagesQueryParts.Selects
		cveResponses, err = pgSearch.RunSelectRequestForSchema[ImageCVEQueryResponse](reportGenCtx, rg.db,
			deployedImagesQueryParts.Schema, query)
		if err != nil {
			return nil, errors.Wrap(err, "Failed to collect report data for deployed images")
		}
		numDeployedImageResults = len(cveResponses)
	}

	numWatchedImageResults := 0
	if filterOnImageType(snap.GetVulnReportFilters().GetImageTypes(), storage.VulnerabilityReportFilters_WATCHED) {
		watchedImages, err := rg.getWatchedImages()
		if err != nil {
			return nil, err
		}
		if len(watchedImages) != 0 {
			query := search.ConjunctionQuery(
				search.NewQueryBuilder().AddExactMatches(search.ImageName, watchedImages...).ProtoQuery(),
				cveFilterQuery)
			query.Pagination = watchedImagesQueryParts.Pagination
			query.Selects = watchedImagesQueryParts.Selects
			watchedImageCVEResponses, err := pgSearch.RunSelectRequestForSchema[ImageCVEQueryResponse](reportGenCtx, rg.db,
				watchedImagesQueryParts.Schema, query)
			if err != nil {
				return nil, errors.Wrap(err, "Failed to collect report data for watched images")
			}
			numWatchedImageResults = len(watchedImageCVEResponses)
			cveResponses = append(cveResponses, watchedImageCVEResponses...)
		}
	}

	cveResponses, err = rg.withCVEReferenceLinks(cveResponses)
	if err != nil {
		return nil, err
	}

	return &ReportData{
		CVEResponses:            cveResponses,
		NumDeployedImageResults: numDeployedImageResults,
		NumWatchedImageResults:  numWatchedImageResults,
	}, nil
}

func (rg *reportGeneratorImpl) buildReportQuery(snap *storage.ReportSnapshot,
	collection *storage.ResourceCollection, dataStartTime time.Time) (*common.ReportQuery, error) {
	qb := common.NewVulnReportQueryBuilder(collection, snap.GetVulnReportFilters(), rg.collectionQueryResolver,
		dataStartTime)
	allClusters, err := rg.clusterDatastore.GetClusters(reportGenCtx)
	if err != nil {
		return nil, errors.Wrap(err, "error fetching clusters to build report query")
	}
	allNamespaces, err := rg.namespaceDatastore.GetAllNamespaces(reportGenCtx)
	if err != nil {
		return nil, errors.Wrap(err, "error fetching namespaces to build report query")
	}
	rQuery, err := qb.BuildQuery(reportGenCtx, allClusters, allNamespaces)
	if err != nil {
		return nil, errors.Wrap(err, "error building report query")
	}
	return rQuery, nil
}

/* Utility Functions */

func (rg *reportGeneratorImpl) retryableSendReportResults(reportNotifier notifiers.ReportNotifier, mailingList []string,
	zippedCSVData *bytes.Buffer, emailSubject, emailBody, baseFilename string) error {
	return retry.WithRetry(func() error {
		return reportNotifier.ReportNotify(reportGenCtx, zippedCSVData, mailingList, emailSubject, emailBody, baseFilename)
	},
		retry.OnlyRetryableErrors(),
		retry.Tries(3),
		retry.BetweenAttempts(func(previousAttempt int) {
			wait := time.Duration(previousAttempt * previousAttempt * 100)
			time.Sleep(wait * time.Millisecond)
		}),
	)
}

func (rg *reportGeneratorImpl) lastSuccessfulScheduledReportTime(snap *storage.ReportSnapshot) (time.Time, error) {
	query := search.NewQueryBuilder().
		AddExactMatches(search.ReportConfigID, snap.GetReportConfigurationId()).
		AddExactMatches(search.ReportRequestType, storage.ReportStatus_SCHEDULED.String()).
		AddExactMatches(search.ReportState, storage.ReportStatus_DELIVERED.String()).
		WithPagination(search.NewPagination().
			AddSortOption(search.NewSortOption(search.ReportCompletionTime).Reversed(true)).
			Limit(1)).
		ProtoQuery()
	results, err := rg.reportSnapshotStore.SearchReportSnapshots(reportGenCtx, query)
	if err != nil {
		return time.Time{}, errors.Wrap(err, "Error finding last successful scheduled report time")
	}
	if len(results) > 1 {
		return time.Time{}, errors.Errorf("Received %d records when only one record is expected", len(results))
	}
	if len(results) == 0 {
		return time.Time{}, nil
	}
	completedAt, err := protocompat.ConvertTimestampToTimeOrError(results[0].GetReportStatus().GetCompletedAt())
	if err != nil {
		return time.Time{}, err
	}
	return completedAt, nil
}

func (rg *reportGeneratorImpl) getWatchedImages() ([]string, error) {
	watched, err := rg.watchedImageDatastore.GetAllWatchedImages(reportGenCtx)
	if err != nil {
		return nil, err
	}
	results := make([]string, 0, len(watched))
	for _, img := range watched {
		results = append(results, img.GetName())
	}
	return results, nil
}

func (rg *reportGeneratorImpl) withCVEReferenceLinks(imageCVEResponses []*ImageCVEQueryResponse) ([]*ImageCVEQueryResponse, error) {
	cveIDs := set.NewStringSet()
	for _, res := range imageCVEResponses {
		if res.GetCVEID() != "" {
			cveIDs.Add(res.GetCVEID())
		}
	}

	var cves []ImageCVEInterface
	if features.FlattenCVEData.Enabled() {
		imageCVEV2, err := rg.imageCVE2Datastore.GetBatch(reportGenCtx, cveIDs.AsSlice())
		if err != nil {
			return nil, err
		}
		for _, v2 := range imageCVEV2 {
			cves = append(cves, v2)
		}
	} else {
		imageCVE, err := rg.imageCVEDatastore.GetBatch(reportGenCtx, cveIDs.AsSlice())
		if err != nil {
			return nil, err
		}
		for _, v2 := range imageCVE {
			cves = append(cves, v2)
		}

	}

	cveRefLinks := make(map[string]string)
	for _, cve := range cves {
		cveRefLinks[cve.GetId()] = cve.GetCveBaseInfo().GetLink()
	}

	for _, res := range imageCVEResponses {
		if link, ok := cveRefLinks[res.GetCVEID()]; ok {
			res.Link = link
		}
	}
	return imageCVEResponses, nil
}

func (rg *reportGeneratorImpl) updateReportStatus(snapshot *storage.ReportSnapshot, status storage.ReportStatus_RunState) error {
	snapshot.ReportStatus.RunState = status
	return rg.reportSnapshotStore.UpdateReportSnapshot(reportGenCtx, snapshot)
}

func (rg *reportGeneratorImpl) logAndUpsertError(reportErr error, req *ReportRequest) {
	if req.ReportSnapshot == nil || req.ReportSnapshot.ReportStatus == nil {
		utils.Should(errors.New("Request does not have non-nil report snapshot with a non-nil report status"))
		return
	}
	if reportErr != nil {
		log.Errorf("Error while running report for config '%s': %s", req.ReportSnapshot.GetName(), reportErr)
		req.ReportSnapshot.ReportStatus.ErrorMsg = reportErr.Error()
	}
	req.ReportSnapshot.ReportStatus.CompletedAt = protocompat.TimestampNow()
	err := rg.updateReportStatus(req.ReportSnapshot, storage.ReportStatus_FAILURE)

	if err != nil {
		log.Errorf("Error changing report status to FAILURE for report config '%s', report ID '%s': %s",
			req.ReportSnapshot.GetName(), req.ReportSnapshot.GetReportId(), err)
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

func selectSchema() *walker.Schema {
	if features.FlattenCVEData.Enabled() {
		return pkgSchema.ImageCvesV2Schema
	}
	return pkgSchema.ImageCvesSchema
}
