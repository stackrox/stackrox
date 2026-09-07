package reportgenerator

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/csv"
	"io"
	"slices"
	"time"

	"github.com/pkg/errors"
	blobDS "github.com/stackrox/rox/central/blob/datastore"
	clusterDS "github.com/stackrox/rox/central/cluster/datastore"
	"github.com/stackrox/rox/central/convert/storagetoeffectiveaccessscope"
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
	"github.com/stackrox/rox/pkg/env"
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
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/effectiveaccessscope"
	"github.com/stackrox/rox/pkg/search"
	pgSearch "github.com/stackrox/rox/pkg/search/postgres"
	"github.com/stackrox/rox/pkg/set"
)

var (
	log = logging.LoggerForModule()

	reportGenCtx = resolvers.SetAuthorizerOverride(loaders.WithLoaderContext(sac.WithAllAccess(context.Background())), allow.Anonymous())

	deployedImagesQueryParts = &ReportQueryParts{
		Schema:  selectSchema(),
		Selects: getSelectsDeployedImages(),
		Pagination: search.NewPagination().
			Limit(int32(env.ReportMaxRows.IntegerSetting())).
			Offset(int32(0)).
			AddSortOption(search.NewSortOption(search.Cluster)).
			AddSortOption(search.NewSortOption(search.Namespace)).Proto(),
	}

	watchedImagesQueryParts = &ReportQueryParts{
		Schema:  selectSchema(),
		Selects: getSelectsWatchedImages(),
		Pagination: search.NewPagination().
			Limit(int32(env.ReportMaxRows.IntegerSetting())).
			Offset(int32(0)).
			AddSortOption(search.NewSortOption(search.ImageName)).Proto(),
	}
	cursorBatchSize = env.PostgresDefaultCursorBatchSize.IntegerSetting()
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
	imageCVE2Datastore      imageCVE2DS.DataStore
	db                      postgres.DB
}

type ImageCVEInterface interface {
	GetId() string
	GetCveBaseInfo() *storage.CVEInfo
}

func (rg *reportGeneratorImpl) ProcessReportRequest(ctx context.Context, req *ReportRequest) {
	ctx = resolvers.SetAuthorizerOverride(loaders.WithLoaderContext(sac.WithAllAccess(ctx)), allow.Anonymous())

	// First do some basic validation checks on the request.
	err := ValidateReportRequest(req)
	if err != nil {
		rg.logAndUpsertError(ctx, errors.Wrap(err, "Invalid report request"), req)
		return
	}

	if req.ReportSnapshot.GetVulnReportFilters() != nil {
		if req.ReportSnapshot.GetVulnReportFilters().GetSinceLastSentScheduledReport() {
			req.DataStartTime, err = rg.lastSuccessfulScheduledReportTime(req.ReportSnapshot)
			if err != nil {
				rg.logAndUpsertError(ctx, errors.Wrap(err, "Error finding last successful scheduled report time"), req)
				return
			}
		} else if req.ReportSnapshot.GetVulnReportFilters().GetSinceStartDate() != nil {
			sinceStartDate := req.ReportSnapshot.GetVulnReportFilters().GetSinceStartDate()
			req.DataStartTime, err = protocompat.ConvertTimestampToTimeOrError(sinceStartDate)
			if err != nil {
				rg.logAndUpsertError(ctx, errors.Wrap(err, "Error finding last successful scheduled report time"), req)
				return
			}
		}
	}

	// Change report status to PREPARING
	err = rg.updateReportStatus(req.ReportSnapshot, storage.ReportStatus_PREPARING)
	if err != nil {
		rg.logAndUpsertError(ctx, errors.Wrap(err, "Error changing report status to PREPARING"), req)
		return
	}

	err = rg.generateReportAndNotify(ctx, req)
	if err != nil {
		rg.logAndUpsertError(ctx, err, req)
		return
	}

	if req.ReportSnapshot.GetReportStatus().GetReportNotificationMethod() == storage.ReportStatus_EMAIL {
		err = rg.updateReportStatus(req.ReportSnapshot, storage.ReportStatus_DELIVERED)
		if err != nil {
			rg.logAndUpsertError(ctx, errors.Wrap(err, "Error changing report status to DELIVERED"), req)
		}
	}
}

/* Report generation helper functions */
func (rg *reportGeneratorImpl) generateReportAndNotify(ctx context.Context, req *ReportRequest) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	// Get the results of running the report query
	var err error
	var reportData *ReportData
	if req.ReportSnapshot.GetReportStatus().GetReportNotificationMethod() == storage.ReportStatus_DOWNLOAD {
		if features.VulnerabilityReportStreamingDownload.Enabled() {
			log.Info("Streaming report generation")
			return rg.generateReportTransaction(ctx, req)
		}
		return rg.generateReportInMemoryDownload(ctx, req)
	}

	// EMAIL path: use existing in-memory approach (email attachments have practical size limits)
	reportData, err = rg.getReportDataSQF(ctx, req.ReportSnapshot, req.Collection, req.DataStartTime)
	if err != nil {
		return err
	}

	if err := ctx.Err(); err != nil {
		return err
	}

	// Format results into CSV
	zippedCSVData, err := GenerateCSV(reportData.CVEResponses, req.ReportSnapshot.GetName())
	if err != nil {
		return err
	}

	req.ReportSnapshot.ReportStatus.CompletedAt = protocompat.TimestampNow()
	err = rg.updateReportStatus(req.ReportSnapshot, storage.ReportStatus_GENERATED)
	if err != nil {
		return errors.Wrap(err, "Error changing report status to GENERATED")
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	defaultEmailSubject, err := FormatEmailSubject(defaultEmailSubjectTemplate, req.ReportSnapshot)
	if err != nil {
		return errors.Wrap(err, "Error generating email subject")
	}
	templateStr := defaultEmailBodyTemplate
	if reportData.NumDeployedImageResults == 0 && reportData.NumWatchedImageResults == 0 {
		zippedCSVData = nil
		templateStr = defaultNoVulnsEmailBodyTemplate
	}

	defaultEmailBody, err := FormatEmailBody(templateStr)
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
		emailBodyWithConfigDetails := AddReportConfigDetails(emailBody, configDetailsHTML)
		reportName := req.ReportSnapshot.GetName()
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
	return nil
}

// generateReportInMemoryDownload accumulates report data in memory, builds the CSV/ZIP, and stores it in blob storage.
func (rg *reportGeneratorImpl) generateReportInMemoryDownload(ctx context.Context, req *ReportRequest) error {
	snap := req.ReportSnapshot

	var reportData *ReportData
	var err error
	if snap.GetViewBasedVulnReportFilters() != nil {
		reportData, err = rg.getReportDataViewBased(ctx, snap)
	} else {
		reportData, err = rg.getReportDataSQF(ctx, snap, req.Collection, req.DataStartTime)
	}
	if err != nil {
		return err
	}

	zippedCSVData, err := GenerateCSV(reportData.CVEResponses, snap.GetName())
	if err != nil {
		return err
	}

	parentDir := snap.GetReportConfigurationId()
	if snap.GetVulnReportFilters() == nil {
		parentDir = "view-based-report"
	}
	if err := rg.saveReportData(ctx, parentDir, snap.GetReportId(), zippedCSVData); err != nil {
		return errors.Wrap(err, "error saving report to blob store")
	}

	snap.ReportStatus.CompletedAt = protocompat.TimestampNow()
	if err := rg.updateReportStatus(snap, storage.ReportStatus_GENERATED); err != nil {
		return errors.Wrap(err, "Error changing report status to GENERATED")
	}
	return nil
}

type querySpec struct {
	schema *walker.Schema
	query  *v1.Query
}

// buildReportQueries constructs the cursor queries for a report request.
func (rg *reportGeneratorImpl) buildReportQueries(ctx context.Context, req *ReportRequest) ([]querySpec, error) {
	var queries []querySpec
	snap := req.ReportSnapshot

	if snap.GetVulnReportFilters() != nil {
		rQuery, err := rg.buildReportQuery(ctx, snap, req.Collection, req.DataStartTime)
		if err != nil {
			return nil, err
		}
		cveFilterQuery, err := search.ParseQuery(rQuery.CveFieldsQuery, search.MatchAllIfEmpty())
		if err != nil {
			return nil, err
		}
		if slices.Contains(snap.GetVulnReportFilters().GetImageTypes(), storage.VulnerabilityReportFilters_DEPLOYED) {
			q := search.ConjunctionQuery(rQuery.DeploymentsQuery, cveFilterQuery)
			q.Pagination = deployedImagesQueryParts.Pagination
			q.Selects = deployedImagesQueryParts.Selects
			queries = append(queries, querySpec{schema: deployedImagesQueryParts.Schema, query: q})
		}
		if slices.Contains(snap.GetVulnReportFilters().GetImageTypes(), storage.VulnerabilityReportFilters_WATCHED) {
			watchedImages, err := rg.getWatchedImages(ctx)
			if err != nil {
				return nil, err
			}
			if len(watchedImages) != 0 {
				q := search.ConjunctionQuery(
					search.NewQueryBuilder().AddExactMatches(search.ImageName, watchedImages...).ProtoQuery(),
					cveFilterQuery)
				q.Pagination = watchedImagesQueryParts.Pagination
				q.Selects = watchedImagesQueryParts.Selects
				queries = append(queries, querySpec{schema: watchedImagesQueryParts.Schema, query: q})
			}
		}
	}
	if snap.GetViewBasedVulnReportFilters() != nil {
		watchedImages, err := rg.getWatchedImages(ctx)
		if err != nil {
			return nil, err
		}
		vbQuery, err := rg.buildReportQueryViewBased(ctx, snap, watchedImages)
		if err != nil {
			return nil, err
		}
		vbQuery.DeployedImagesQuery.Pagination = deployedImagesQueryParts.Pagination
		vbQuery.DeployedImagesQuery.Selects = deployedImagesQueryParts.Selects
		queries = append(queries, querySpec{schema: deployedImagesQueryParts.Schema, query: vbQuery.DeployedImagesQuery})

		if len(watchedImages) != 0 {
			vbQuery.WatchedImagesQuery.Pagination = watchedImagesQueryParts.Pagination
			vbQuery.WatchedImagesQuery.Selects = watchedImagesQueryParts.Selects
			queries = append(queries, querySpec{schema: watchedImagesQueryParts.Schema, query: vbQuery.WatchedImagesQuery})
		}
	}
	return queries, nil
}

// generateReportTransaction runs all report operations — cursor reads, CVE
// lookups, CSV/ZIP generation, and blob store write — within a single database
// transaction. The writeFn callback writes directly to a PostgreSQL large
// object, so memory usage stays constant regardless of report size.
//
// Connection usage: 1 connection. All operations (cursor FETCH, CVE lookups,
// large object writes, metadata upsert) share the same transaction on a single
// goroutine, so there are no concurrent-tx-access issues.
func (rg *reportGeneratorImpl) generateReportTransaction(ctx context.Context, req *ReportRequest) error {
	snap := req.ReportSnapshot
	queries, err := rg.buildReportQueries(ctx, req)
	if err != nil {
		return err
	}

	tx, err := rg.db.Begin(ctx)
	if err != nil {
		return errors.Wrap(err, "starting report transaction")
	}
	committed := false
	defer func() {
		if !committed {
			if rbErr := tx.Rollback(context.Background()); rbErr != nil {
				log.Errorf("failed to rollback report transaction: %v", rbErr)
			}
		}
	}()
	txCtx := postgres.ContextWithTx(ctx, tx)

	parentDir := snap.GetReportConfigurationId()
	if snap.GetVulnReportFilters() == nil {
		parentDir = "view-based-report"
	}
	blobPath := common.GetReportBlobPath(parentDir, snap.GetReportId())
	blob := &storage.Blob{
		Name:         blobPath,
		LastUpdated:  protocompat.TimestampNow(),
		ModifiedTime: protocompat.TimestampNow(),
		Length:       -1,
	}

	//  cveRefLinksCache is a map[string]string that maps CVE ID -> reference URL.
	//  It caches the "Reference" link for each CVE (the external URL pointing to the CVE advisory, e.g., on NVD).
	cveRefLinksCache := make(map[string]string)
	rowCount := 0

	err = rg.blobStore.UpsertWithWriter(txCtx, tx, blob, func(w io.Writer) error {
		zipWriter := zip.NewWriter(w)
		zipEntry, err := zipWriter.Create(csvReportName(snap.GetName()))
		if err != nil {
			return errors.Wrap(err, "creating zip entry")
		}
		csvW := csv.NewWriter(zipEntry)
		csvW.UseCRLF = true
		if err := csvW.Write(formatCol()); err != nil {
			return errors.Wrap(err, "writing CSV header")
		}

		for _, qs := range queries {
			if err := rg.streamQueryToCSV(txCtx, qs.schema, qs.query, csvW, cveRefLinksCache, &rowCount); err != nil {
				return err
			}
		}

		csvW.Flush()
		if err := csvW.Error(); err != nil {
			return errors.Wrap(err, "flushing CSV writer")
		}
		return zipWriter.Close()
	})
	if err != nil {
		return errors.Wrap(err, "error streaming report to blob store")
	}

	if err := tx.Commit(context.Background()); err != nil {
		return errors.Wrap(err, "committing report transaction")
	}
	committed = true

	snap.ReportStatus.CompletedAt = protocompat.TimestampNow()
	if err := rg.updateReportStatus(snap, storage.ReportStatus_GENERATED); err != nil {
		return errors.Wrap(err, "Error changing report status to GENERATED")
	}
	return nil
}

// streamQueryToCSV runs a cursor-based query and streams each row directly through CSV formatting
// to the provided csv.Writer. CVE reference links are resolved incrementally in batches and cached.
func (rg *reportGeneratorImpl) streamQueryToCSV(
	ctx context.Context,
	schema *walker.Schema,
	query *v1.Query,
	csvW *csv.Writer,
	cveRefLinksCache map[string]string,
	rowCount *int,
) error {
	var batch []*ImageCVEQueryResponse

	flushBatch := func() error {
		unseenIDs := set.NewStringSet()
		for _, r := range batch {
			id := r.GetCVEID()
			if id != "" {
				if _, cached := cveRefLinksCache[id]; !cached {
					unseenIDs.Add(id)
				}
			}
		}
		if unseenIDs.Cardinality() > 0 {
			cves, err := rg.imageCVE2Datastore.GetBatch(ctx, unseenIDs.AsSlice())
			if err != nil {
				return errors.Wrap(err, "fetching CVE reference links")
			}
			for _, cve := range cves {
				cveRefLinksCache[cve.GetId()] = cve.GetCveBaseInfo().GetLink()
			}
		}
		for _, r := range batch {
			if link, ok := cveRefLinksCache[r.GetCVEID()]; ok {
				r.Link = link
			}
			if err := csvW.Write(formatCSVRow(r)); err != nil {
				return err
			}
		}
		batch = batch[:0]
		return nil
	}

	err := pgSearch.RunSelectCursorForSchemaFn[ImageCVEQueryResponse](
		ctx, rg.db, schema, query,
		func(r *ImageCVEQueryResponse) error {
			batch = append(batch, r)
			*rowCount++
			if len(batch) >= cursorBatchSize {
				return flushBatch()
			}
			return nil
		})
	if err != nil {
		return err
	}

	if err := ctx.Err(); err != nil {
		return err
	}

	if len(batch) > 0 {
		return flushBatch()
	}
	return nil
}

func (rg *reportGeneratorImpl) saveReportData(ctx context.Context, configID, reportID string, data *bytes.Buffer) error {
	return SaveReportData(ctx, rg.blobStore, configID, reportID, data)
}

func (rg *reportGeneratorImpl) getReportDataSQF(ctx context.Context, snap *storage.ReportSnapshot, collection *storage.ResourceCollection,
	dataStartTime time.Time) (*ReportData, error) {
	rQuery, err := rg.buildReportQuery(ctx, snap, collection, dataStartTime)
	if err != nil {
		return nil, err
	}

	cveFilterQuery, err := search.ParseQuery(rQuery.CveFieldsQuery, search.MatchAllIfEmpty())
	if err != nil {
		return nil, err
	}

	numDeployedImageResults := 0
	var cveResponses []*ImageCVEQueryResponse
	if slices.Contains(snap.GetVulnReportFilters().GetImageTypes(), storage.VulnerabilityReportFilters_DEPLOYED) {
		query := search.ConjunctionQuery(rQuery.DeploymentsQuery, cveFilterQuery)
		query.Pagination = deployedImagesQueryParts.Pagination
		query.Selects = deployedImagesQueryParts.Selects
		err = pgSearch.RunSelectRequestForSchemaFn[ImageCVEQueryResponse](ctx, rg.db,
			deployedImagesQueryParts.Schema, query, func(r *ImageCVEQueryResponse) error {
				cveResponses = append(cveResponses, r)
				numDeployedImageResults++
				return nil
			})
		if err != nil {
			return nil, errors.Wrap(err, "Failed to collect report data for deployed images")
		}
	}

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	numWatchedImageResults := 0
	if slices.Contains(snap.GetVulnReportFilters().GetImageTypes(), storage.VulnerabilityReportFilters_WATCHED) {
		watchedImages, err := rg.getWatchedImages(ctx)
		if err != nil {
			return nil, err
		}
		if len(watchedImages) != 0 {
			query := search.ConjunctionQuery(
				search.NewQueryBuilder().AddExactMatches(search.ImageName, watchedImages...).ProtoQuery(),
				cveFilterQuery)
			query.Pagination = watchedImagesQueryParts.Pagination
			query.Selects = watchedImagesQueryParts.Selects
			err := pgSearch.RunSelectRequestForSchemaFn[ImageCVEQueryResponse](ctx, rg.db,
				watchedImagesQueryParts.Schema, query, func(r *ImageCVEQueryResponse) error {
					cveResponses = append(cveResponses, r)
					numWatchedImageResults++
					return nil
				})
			if err != nil {
				return nil, errors.Wrap(err, "Failed to collect report data for watched images")
			}
		}
	}

	cveResponses, err = rg.withCVEReferenceLinks(ctx, cveResponses)
	if err != nil {
		return nil, err
	}

	return &ReportData{
		CVEResponses:            cveResponses,
		NumDeployedImageResults: numDeployedImageResults,
		NumWatchedImageResults:  numWatchedImageResults,
	}, nil
}

func (rg *reportGeneratorImpl) getReportDataViewBased(ctx context.Context, snap *storage.ReportSnapshot) (*ReportData, error) {
	watchedImages, err := rg.getWatchedImages(ctx)
	if err != nil {
		return nil, err
	}
	query, err := rg.buildReportQueryViewBased(ctx, snap, watchedImages)
	if err != nil {
		return nil, err
	}

	numDeployedImageResults := 0
	var cveResponses []*ImageCVEQueryResponse

	query.DeployedImagesQuery.Pagination = deployedImagesQueryParts.Pagination
	query.DeployedImagesQuery.Selects = deployedImagesQueryParts.Selects
	err = pgSearch.RunSelectRequestForSchemaFn[ImageCVEQueryResponse](ctx, rg.db,
		deployedImagesQueryParts.Schema, query.DeployedImagesQuery, func(r *ImageCVEQueryResponse) error {
			cveResponses = append(cveResponses, r)
			return nil
		})
	if err != nil {
		return nil, errors.Wrap(err, "Failed to collect report data for deployed images")
	}
	numDeployedImageResults = len(cveResponses)

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	numWatchedImageResults := 0

	if len(watchedImages) != 0 {
		query.WatchedImagesQuery.Pagination = watchedImagesQueryParts.Pagination
		query.WatchedImagesQuery.Selects = watchedImagesQueryParts.Selects
		err := pgSearch.RunSelectRequestForSchemaFn[ImageCVEQueryResponse](ctx, rg.db,
			watchedImagesQueryParts.Schema, query.WatchedImagesQuery, func(r *ImageCVEQueryResponse) error {
				cveResponses = append(cveResponses, r)
				numWatchedImageResults++
				return nil
			})
		if err != nil {
			return nil, errors.Wrap(err, "Failed to collect report data for watched images")
		}
	}

	cveResponses, err = rg.withCVEReferenceLinks(ctx, cveResponses)
	if err != nil {
		return nil, err
	}

	return &ReportData{
		CVEResponses:            cveResponses,
		NumDeployedImageResults: numDeployedImageResults,
		NumWatchedImageResults:  numWatchedImageResults,
	}, nil

}

func (rg *reportGeneratorImpl) getClustersAndNamespacesForSAC(ctx context.Context) ([]effectiveaccessscope.Cluster, []effectiveaccessscope.Namespace, error) {
	allClusters, err := rg.clusterDatastore.GetClusters(ctx)
	if err != nil {
		return nil, nil, errors.Wrap(err, "error fetching clusters to build report query")
	}
	sacClusters := storagetoeffectiveaccessscope.Clusters(allClusters)
	allNamespaces, err := rg.namespaceDatastore.GetAllNamespaces(ctx)
	if err != nil {
		return nil, nil, errors.Wrap(err, "error fetching namespaces to build report query")
	}
	sacNamespaces := storagetoeffectiveaccessscope.Namespaces(allNamespaces)
	return sacClusters, sacNamespaces, nil
}

func (rg *reportGeneratorImpl) buildReportQueryViewBased(ctx context.Context, snap *storage.ReportSnapshot, watchedImages []string) (*common.ReportQueryViewBased, error) {
	qb := common.NewVulnReportQueryBuilderViewBased(snap.GetViewBasedVulnReportFilters())
	allClusters, allNamespaces, err := rg.getClustersAndNamespacesForSAC(ctx)
	if err != nil {
		return nil, err
	}
	rQuery, err := qb.BuildQueryViewBased(allClusters, allNamespaces, watchedImages)
	if err != nil {
		return nil, errors.Wrap(err, "error building report query")
	}
	return rQuery, nil
}

func (rg *reportGeneratorImpl) buildReportQuery(ctx context.Context, snap *storage.ReportSnapshot,
	collection *storage.ResourceCollection, dataStartTime time.Time) (*common.ReportQuery, error) {
	qb := common.NewVulnReportQueryBuilder(collection, snap.GetResourceScope().GetEntityScope(), snap.GetVulnReportFilters(), rg.collectionQueryResolver,
		dataStartTime)
	allClusters, allNamespaces, err := rg.getClustersAndNamespacesForSAC(ctx)
	if err != nil {
		return nil, err
	}
	rQuery, err := qb.BuildQuery(ctx, allClusters, allNamespaces)
	if err != nil {
		return nil, errors.Wrap(err, "error building report query")
	}
	return rQuery, nil
}

/* Utility Functions */

func (rg *reportGeneratorImpl) retryableSendReportResults(reportNotifier notifiers.ReportNotifier, mailingList []string,
	zippedCSVData *bytes.Buffer, emailSubject, emailBody, baseFilename string) error {
	return RetryableSendReportResults(reportGenCtx, reportNotifier, mailingList,
		zippedCSVData, emailSubject, emailBody, baseFilename)
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

func (rg *reportGeneratorImpl) getWatchedImages(ctx context.Context) ([]string, error) {
	watched, err := rg.watchedImageDatastore.GetAllWatchedImages(ctx)
	if err != nil {
		return nil, err
	}
	results := make([]string, 0, len(watched))
	for _, img := range watched {
		results = append(results, img.GetName())
	}
	return results, nil
}

func (rg *reportGeneratorImpl) withCVEReferenceLinks(ctx context.Context, imageCVEResponses []*ImageCVEQueryResponse) ([]*ImageCVEQueryResponse, error) {
	cveIDs := set.NewStringSet()
	for _, res := range imageCVEResponses {
		if res.GetCVEID() != "" {
			cveIDs.Add(res.GetCVEID())
		}
	}

	var cves []ImageCVEInterface
	imageCVEV2, err := rg.imageCVE2Datastore.GetBatch(ctx, cveIDs.AsSlice())
	if err != nil {
		return nil, err
	}
	for _, v2 := range imageCVEV2 {
		cves = append(cves, v2)
	}

	// cveRefLinks is a map[string]string that maps CVE ID -> reference URL
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
	return UpdateReportStatus(reportGenCtx, rg.reportSnapshotStore, snapshot, status)
}

func (rg *reportGeneratorImpl) logAndUpsertError(ctx context.Context, reportErr error, req *ReportRequest) {
	LogAndUpsertError(ctx, reportGenCtx, rg.reportSnapshotStore, reportErr, req)
}

func selectSchema() *walker.Schema {
	return pkgSchema.ImageCvesV2Schema
}

func getSelectsWatchedImages() []*v1.QuerySelect {
	ret := []*v1.QuerySelect{
		search.NewQuerySelect(search.ImageName).Proto(),
		search.NewQuerySelect(search.Component).Proto(),
		search.NewQuerySelect(search.ComponentVersion).Proto(),
		search.NewQuerySelect(search.CVEID).Proto(),
		search.NewQuerySelect(search.CVE).Proto(),
		search.NewQuerySelect(search.Fixable).Proto(),
		search.NewQuerySelect(search.FixedBy).Proto(),
		search.NewQuerySelect(search.Severity).Proto(),
		search.NewQuerySelect(search.CVSS).Proto(),
		search.NewQuerySelect(search.NVDCVSS).Proto(),
		search.NewQuerySelect(search.FirstImageOccurrenceTimestamp).Proto(),
		search.NewQuerySelect(search.EPSSProbablity).Proto(),
		search.NewQuerySelect(search.CisaKev).Proto(),
		search.NewQuerySelect(search.AdvisoryName).Proto(),
		search.NewQuerySelect(search.AdvisoryLink).Proto(),
		search.NewQuerySelect(search.CVEOrigin).Proto(),
	}
	return ret
}

func getSelectsDeployedImages() []*v1.QuerySelect {
	ret := []*v1.QuerySelect{
		search.NewQuerySelect(search.ImageName).Proto(),
		search.NewQuerySelect(search.Component).Proto(),
		search.NewQuerySelect(search.ComponentVersion).Proto(),
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
		search.NewQuerySelect(search.CisaKev).Proto(),
		search.NewQuerySelect(search.AdvisoryName).Proto(),
		search.NewQuerySelect(search.AdvisoryLink).Proto(),
		search.NewQuerySelect(search.CVEOrigin).Proto(),
	}
	return ret
}
