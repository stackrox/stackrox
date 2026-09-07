package node

import (
	"bytes"
	"context"

	"github.com/pkg/errors"
	clusterDS "github.com/stackrox/rox/central/cluster/datastore"
	"github.com/stackrox/rox/central/convert/storagetoeffectiveaccessscope"
	nodeCVEDS "github.com/stackrox/rox/central/cve/node/datastore"
	namespaceDS "github.com/stackrox/rox/central/namespace/datastore"
	reportGen "github.com/stackrox/rox/central/reports/scheduler/v2/reportgenerator"
	reportSnapshotDS "github.com/stackrox/rox/central/reports/snapshot/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/notifier"
	"github.com/stackrox/rox/pkg/notifiers"
	"github.com/stackrox/rox/pkg/postgres"
	pkgSchema "github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/effectiveaccessscope"
	"github.com/stackrox/rox/pkg/search"
	pgSearch "github.com/stackrox/rox/pkg/search/postgres"
	"github.com/stackrox/rox/pkg/set"

	blobDS "github.com/stackrox/rox/central/blob/datastore"
)

var (
	nodeReportGenCtx = sac.WithAllAccess(context.Background())

	nodeQueryParts = &reportGen.ReportQueryParts{
		Schema: pkgSchema.NodeCvesSchema,
		Selects: []*v1.QuerySelect{
			search.NewQuerySelect(search.Cluster).Proto(),
			search.NewQuerySelect(search.Node).Proto(),
			search.NewQuerySelect(search.Component).Proto(),
			search.NewQuerySelect(search.ComponentVersion).Proto(),
			search.NewQuerySelect(search.CVEID).Proto(),
			search.NewQuerySelect(search.CVE).Proto(),
			search.NewQuerySelect(search.Fixable).Proto(),
			search.NewQuerySelect(search.FixedBy).Proto(),
			search.NewQuerySelect(search.Severity).Proto(),
			search.NewQuerySelect(search.CVSS).Proto(),
			search.NewQuerySelect(search.CVECreatedTime).Proto(),
		},
		Pagination: search.NewPagination().
			Limit(int32(env.ReportMaxRows.IntegerSetting())).
			Offset(0).
			AddSortOption(search.NewSortOption(search.Cluster)).
			AddSortOption(search.NewSortOption(search.Node)).
			Proto(),
	}
)

type nodeReportGeneratorImpl struct {
	reportSnapshotStore   reportSnapshotDS.DataStore
	notificationProcessor notifier.Processor
	blobStore             blobDS.Datastore
	clusterDatastore      clusterDS.DataStore
	namespaceDatastore    namespaceDS.DataStore
	nodeCVEDatastore      nodeCVEDS.DataStore
	db                    postgres.DB
}

func (rg *nodeReportGeneratorImpl) ProcessReportRequest(ctx context.Context, req *reportGen.ReportRequest) {
	ctx = sac.WithAllAccess(ctx)

	if err := reportGen.ValidateReportRequest(req); err != nil {
		reportGen.LogAndUpsertError(ctx, nodeReportGenCtx, rg.reportSnapshotStore, errors.Wrap(err, "Invalid report request"), req)
		return
	}

	if err := reportGen.UpdateReportStatus(nodeReportGenCtx, rg.reportSnapshotStore, req.ReportSnapshot, storage.ReportStatus_PREPARING); err != nil {
		reportGen.LogAndUpsertError(ctx, nodeReportGenCtx, rg.reportSnapshotStore, errors.Wrap(err, "Error changing report status to PREPARING"), req)
		return
	}

	if err := rg.generateReportAndNotify(ctx, req); err != nil {
		reportGen.LogAndUpsertError(ctx, nodeReportGenCtx, rg.reportSnapshotStore, err, req)
		return
	}

	if req.ReportSnapshot.GetReportStatus().GetReportNotificationMethod() == storage.ReportStatus_EMAIL {
		if err := reportGen.UpdateReportStatus(nodeReportGenCtx, rg.reportSnapshotStore, req.ReportSnapshot, storage.ReportStatus_DELIVERED); err != nil {
			reportGen.LogAndUpsertError(ctx, nodeReportGenCtx, rg.reportSnapshotStore, errors.Wrap(err, "Error changing report status to DELIVERED"), req)
		}
	}
}

func (rg *nodeReportGeneratorImpl) generateReportAndNotify(ctx context.Context, req *reportGen.ReportRequest) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	cveResponses, numResults, err := rg.getReportData(ctx, req.ReportSnapshot)
	if err != nil {
		return err
	}

	if err := ctx.Err(); err != nil {
		return err
	}

	zippedCSVData, err := generateCSV(cveResponses, req.ReportSnapshot.GetName())
	if err != nil {
		return err
	}

	req.ReportSnapshot.ReportStatus.CompletedAt = protocompat.TimestampNow()
	if err := reportGen.UpdateReportStatus(nodeReportGenCtx, rg.reportSnapshotStore, req.ReportSnapshot, storage.ReportStatus_GENERATED); err != nil {
		return errors.Wrap(err, "Error changing report status to GENERATED")
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	switch req.ReportSnapshot.GetReportStatus().GetReportNotificationMethod() {
	case storage.ReportStatus_DOWNLOAD:
		parentDir := req.ReportSnapshot.GetReportConfigurationId()
		if req.ReportSnapshot.GetReportStatus().GetReportRequestType() == storage.ReportStatus_VIEW_BASED {
			parentDir = "view-based-report"
		}
		if err := reportGen.SaveReportData(nodeReportGenCtx, rg.blobStore, parentDir, req.ReportSnapshot.GetReportId(), zippedCSVData); err != nil {
			return errors.Wrap(err, "error persisting blob")
		}
	case storage.ReportStatus_EMAIL:
		return rg.sendEmailNotification(req, zippedCSVData, numResults)
	}
	return nil
}

func (rg *nodeReportGeneratorImpl) sendEmailNotification(req *reportGen.ReportRequest, zippedCSVData *bytes.Buffer, numResults int) error {
	defaultEmailSubject, err := reportGen.FormatEmailSubject(reportGen.NodeDefaultEmailSubjectTemplate, req.ReportSnapshot)
	if err != nil {
		return errors.Wrap(err, "Error generating email subject")
	}

	templateStr := reportGen.NodeDefaultEmailBodyTemplate
	if numResults == 0 {
		zippedCSVData = nil
		templateStr = reportGen.NodeDefaultNoVulnsEmailBodyTemplate
	}

	defaultEmailBody, err := reportGen.FormatEmailBody(templateStr)
	if err != nil {
		return errors.Wrap(err, "Error generating email body")
	}

	configDetailsHTML, err := reportGen.FormatNodeReportConfigDetails(req.ReportSnapshot, numResults)
	if err != nil {
		return errors.Wrap(err, "Error adding report config details")
	}

	errorList := errorhelpers.NewErrorList("Error sending email notifications: ")
	for _, notifierSnap := range req.ReportSnapshot.GetNotifiers() {
		nf := rg.notificationProcessor.GetNotifier(nodeReportGenCtx, notifierSnap.GetEmailConfig().GetNotifierId())
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
		emailBodyWithConfigDetails := reportGen.AddReportConfigDetails(emailBody, configDetailsHTML)
		reportName := req.ReportSnapshot.GetName()
		if err := reportGen.RetryableSendReportResults(nodeReportGenCtx, reportNotifier, notifierSnap.GetEmailConfig().GetMailingLists(),
			zippedCSVData, emailSubject, emailBodyWithConfigDetails, reportName); err != nil {
			errorList.AddError(errors.Errorf("Error sending email for notifier '%s': %s",
				notifierSnap.GetEmailConfig().GetNotifierId(), err))
		}
	}
	if !errorList.Empty() {
		return errorList.ToError()
	}
	return nil
}

func (rg *nodeReportGeneratorImpl) getReportData(ctx context.Context, snap *storage.ReportSnapshot) ([]*NodeCVEQueryResponse, int, error) {
	clusters, namespaces, err := rg.getClustersAndNamespacesForSAC(ctx)
	if err != nil {
		return nil, 0, err
	}

	var entityScope *storage.EntityScope
	if snap.GetReportStatus().GetReportRequestType() != storage.ReportStatus_VIEW_BASED {
		entityScope = snap.GetResourceScope().GetEntityScope()
	}

	qb := newQueryBuilder(entityScope, snap.GetNodeVulnReportFilters())
	query, err := qb.buildQuery(clusters, namespaces)
	if err != nil {
		return nil, 0, errors.Wrap(err, "error building node report query")
	}

	return rg.executeQuery(ctx, query)
}

func (rg *nodeReportGeneratorImpl) executeQuery(ctx context.Context, query *v1.Query) ([]*NodeCVEQueryResponse, int, error) {
	query.Pagination = nodeQueryParts.Pagination
	query.Selects = nodeQueryParts.Selects

	var cveResponses []*NodeCVEQueryResponse
	cveIDs := set.NewStringSet()
	err := pgSearch.RunSelectRequestForSchemaFn[NodeCVEQueryResponse](ctx, rg.db,
		nodeQueryParts.Schema, query, func(r *NodeCVEQueryResponse) error {
			cveResponses = append(cveResponses, r)
			if r.GetCVEID() != "" {
				cveIDs.Add(r.GetCVEID())
			}
			return nil
		})
	if err != nil {
		return nil, 0, errors.Wrap(err, "Failed to collect node CVE report data")
	}

	cveResponses, err = rg.withCVEReferenceLinks(ctx, cveResponses, cveIDs)
	if err != nil {
		return nil, 0, err
	}

	return cveResponses, len(cveResponses), nil
}

func (rg *nodeReportGeneratorImpl) getClustersAndNamespacesForSAC(ctx context.Context) ([]effectiveaccessscope.Cluster, []effectiveaccessscope.Namespace, error) {
	allClusters, err := rg.clusterDatastore.GetClusters(ctx)
	if err != nil {
		return nil, nil, errors.Wrap(err, "error fetching clusters to build report query")
	}
	allNamespaces, err := rg.namespaceDatastore.GetAllNamespaces(ctx)
	if err != nil {
		return nil, nil, errors.Wrap(err, "error fetching namespaces to build report query")
	}
	return storagetoeffectiveaccessscope.Clusters(allClusters), storagetoeffectiveaccessscope.Namespaces(allNamespaces), nil
}

func (rg *nodeReportGeneratorImpl) withCVEReferenceLinks(ctx context.Context, responses []*NodeCVEQueryResponse, cveIDs set.StringSet) ([]*NodeCVEQueryResponse, error) {
	nodeCVEs, err := rg.nodeCVEDatastore.GetBatch(ctx, cveIDs.AsSlice())
	if err != nil {
		return nil, err
	}

	cveRefLinks := make(map[string]string, len(nodeCVEs))
	for _, cve := range nodeCVEs {
		cveRefLinks[cve.GetId()] = cve.GetCveBaseInfo().GetLink()
	}

	for _, res := range responses {
		if link, ok := cveRefLinks[res.GetCVEID()]; ok {
			res.Link = link
		}
	}
	return responses, nil
}
