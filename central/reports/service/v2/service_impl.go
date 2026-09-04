package v2

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/pkg/errors"
	blobDS "github.com/stackrox/rox/central/blob/datastore"
	notifierDS "github.com/stackrox/rox/central/notifier/datastore"
	"github.com/stackrox/rox/central/reports/common"
	reportConfigDS "github.com/stackrox/rox/central/reports/config/datastore"
	schedulerV2 "github.com/stackrox/rox/central/reports/scheduler/v2"
	reportGen "github.com/stackrox/rox/central/reports/scheduler/v2/reportgenerator"
	snapshotDS "github.com/stackrox/rox/central/reports/snapshot/datastore"
	"github.com/stackrox/rox/central/reports/validation"
	collectionDS "github.com/stackrox/rox/central/resourcecollection/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	apiV2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"github.com/stackrox/rox/pkg/postgres"
	pgNotify "github.com/stackrox/rox/pkg/postgres/notify"
	"github.com/stackrox/rox/pkg/retry"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/paginated"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/stringutils"
	"google.golang.org/grpc"
)

const (
	maxPaginationLimit = 1000
)

var (
	workflowSAC = sac.ForResource(resources.WorkflowAdministration)

	authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		// V2 API authorization
		// TO DO ROX-35954: add view deployment permission to report config APIs
		user.With(permissions.View(resources.WorkflowAdministration), permissions.View(resources.Image)): {
			apiV2.ReportService_ListReportConfigurations_FullMethodName,
			apiV2.ReportService_GetReportConfiguration_FullMethodName,
			apiV2.ReportService_CountReportConfigurations_FullMethodName,
		},
		user.With(permissions.Modify(resources.WorkflowAdministration), permissions.View(resources.Integration), permissions.View(resources.Image)): {
			apiV2.ReportService_PostReportConfiguration_FullMethodName,
			apiV2.ReportService_UpdateReportConfiguration_FullMethodName,
		},
		user.With(permissions.Modify(resources.WorkflowAdministration), permissions.View(resources.Image)): {
			apiV2.ReportService_DeleteReportConfiguration_FullMethodName,
			apiV2.ReportService_RunReport_FullMethodName,
		},
		user.With(permissions.View(resources.WorkflowAdministration), permissions.View(resources.Image)): {
			apiV2.ReportService_GetReportStatus_FullMethodName,
			apiV2.ReportService_GetReportHistory_FullMethodName,
			apiV2.ReportService_GetMyReportHistory_FullMethodName,
		},
		user.With(permissions.View(resources.Image), permissions.View(resources.Deployment)): {
			apiV2.ReportService_GetViewBasedReportHistory_FullMethodName,
			apiV2.ReportService_GetViewBasedMyReportHistory_FullMethodName,
			apiV2.ReportService_PostViewBasedReport_FullMethodName,
		},
		// view permissions are enough if user is deleting a job created by the user
		// TO DO ROX-35954: add view deployment permission
		user.With(permissions.View(resources.Image)): {
			apiV2.ReportService_DeleteReport_FullMethodName,
			apiV2.ReportService_CancelReport_FullMethodName,
		},
	})
)

type serviceImpl struct {
	apiV2.UnimplementedReportServiceServer
	reportConfigStore   reportConfigDS.DataStore
	snapshotDatastore   snapshotDS.DataStore
	collectionDatastore collectionDS.DataStore
	notifierDatastore   notifierDS.DataStore
	scheduler           schedulerV2.Scheduler
	blobStore           blobDS.Datastore
	validator           *validation.Validator
	db                  postgres.DB
}

func (s *serviceImpl) RegisterServiceServer(grpcServer *grpc.Server) {
	apiV2.RegisterReportServiceServer(grpcServer, s)
}

func (s *serviceImpl) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return apiV2.RegisterReportServiceHandler(ctx, mux, conn)
}

func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, authorizer.Authorized(ctx, fullMethodName)
}

func (s *serviceImpl) PostReportConfiguration(ctx context.Context, request *apiV2.ReportConfiguration) (*apiV2.ReportConfiguration, error) {
	creatorID := authn.IdentityFromContextOrNil(ctx)
	if creatorID == nil {
		return nil, errors.New("Could not determine user identity from provided context")
	}

	if request.GetType() == apiV2.ReportConfiguration_NODE_VULNERABILITY {
		return nil, errox.InvalidArgs.New("node vulnerability reports must be created via the node report service")
	}

	if err := s.validator.ValidateReportConfiguration(request); err != nil {
		return nil, errors.Wrap(err, "Validating report configuration")
	}

	creator := &storage.SlimUser{
		Id:   creatorID.UID(),
		Name: stringutils.FirstNonEmpty(creatorID.FullName(), creatorID.FriendlyName()),
	}

	protoReportConfig := s.convertV2ReportConfigurationToProto(request, creator, common.ExtractAccessScopeRules(creatorID))

	id, err := s.reportConfigStore.AddReportConfiguration(ctx, protoReportConfig)
	if err != nil {
		return nil, err
	}

	createdReportConfig, _, err := s.reportConfigStore.GetReportConfiguration(ctx, id)
	if err != nil {
		return nil, err
	}

	if env.CentralWorkerEnabled.BooleanSetting() {
		notifyWithRetry(ctx, s.db, pgNotify.ReportConfigChanged, id)
	} else if err := s.scheduler.UpsertReportSchedule(createdReportConfig); err != nil {
		return nil, err
	}

	resp, err := s.convertProtoReportConfigurationToV2(createdReportConfig)
	if err != nil {
		return nil, errors.Wrap(err, "Report config created, but encountered error generating the response")
	}
	return resp, nil
}

func (s *serviceImpl) UpdateReportConfiguration(ctx context.Context, request *apiV2.ReportConfiguration) (*apiV2.Empty, error) {
	if request.GetId() == "" {
		return nil, errors.Wrap(errox.InvalidArgs, "Report configuration id is required")
	}
	if err := s.validator.ValidateReportConfiguration(request); err != nil {
		return nil, errors.Wrap(err, "Validating report configuration")
	}

	currentConfig, exists, err := s.reportConfigStore.GetReportConfiguration(ctx, request.GetId())
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.Wrapf(errox.NotFound, "report configuration with id '%s' does not exist", request.GetId())
	}
	if err := rejectNodeReportConfiguration(currentConfig); err != nil {
		return nil, err
	}

	query := search.NewQueryBuilder().AddExactMatches(search.ReportConfigID, request.GetId()).AddExactMatches(search.ReportState, storage.ReportStatus_WAITING.String(), storage.ReportStatus_PREPARING.String()).ProtoQuery()
	reportSnapshots, err := s.snapshotDatastore.SearchReportSnapshots(ctx, query)
	if err != nil {
		return nil, err
	}
	slimUser := authn.UserFromContext(ctx)
	for _, reportSnapshot := range reportSnapshots {
		if slimUser.GetId() == reportSnapshot.GetRequester().GetId() {
			return nil, errors.Wrap(errox.InvalidArgs, "User has a report job running for this configuration.")
		}
	}

	var accessScopeRules []*storage.SimpleAccessScope_Rules
	if filters := currentConfig.GetVulnReportFilters(); filters != nil {
		accessScopeRules = filters.GetAccessScopeRules()
	}
	updatedConfig := s.convertV2ReportConfigurationToProto(request, currentConfig.GetCreator(), accessScopeRules)

	err = s.reportConfigStore.UpdateReportConfiguration(ctx, updatedConfig)
	if err != nil {
		return nil, err
	}

	if env.CentralWorkerEnabled.BooleanSetting() {
		notifyWithRetry(ctx, s.db, pgNotify.ReportConfigChanged, updatedConfig.GetId())
	} else if err := s.scheduler.UpsertReportSchedule(updatedConfig); err != nil {
		return nil, err
	}
	return &apiV2.Empty{}, nil
}

func (s *serviceImpl) ListReportConfigurations(ctx context.Context, query *apiV2.RawQuery) (*apiV2.ListReportConfigurationsResponse, error) {
	// Fill in Query.
	parsedQuery, err := search.ParseQuery(query.GetQuery(), search.MatchAllIfEmpty())
	if err != nil {
		return nil, errors.Wrap(errox.InvalidArgs, err.Error())
	}
	parsedQuery = search.ConjunctionQuery(
		parsedQuery,
		search.NewQueryBuilder().AddExactMatches(search.ReportType, storage.ReportConfiguration_VULNERABILITY.String()).ProtoQuery(),
	)
	// Fill in pagination.
	paginated.FillPaginationV2(parsedQuery, query.GetPagination(), maxPaginationLimit)

	reportConfigs, err := s.reportConfigStore.GetReportConfigurations(ctx, parsedQuery)
	if err != nil {
		return nil, errors.Wrap(err, "failed to retrieve report configurations")
	}
	v2Configs := make([]*apiV2.ReportConfiguration, 0, len(reportConfigs))

	for _, config := range reportConfigs {
		converted, err := s.convertProtoReportConfigurationToV2(config)
		if err != nil {
			return nil, errors.Wrapf(err, "Error converting storage report configuration with id %s to response", config.GetId())
		}
		v2Configs = append(v2Configs, converted)
	}
	return &apiV2.ListReportConfigurationsResponse{ReportConfigs: v2Configs}, nil
}

func (s *serviceImpl) GetReportConfiguration(ctx context.Context, req *apiV2.ResourceByID) (*apiV2.ReportConfiguration, error) {
	if req.GetId() == "" {
		return nil, errors.Wrap(errox.InvalidArgs, "Report configuration id is required")
	}
	config, exists, err := s.reportConfigStore.GetReportConfiguration(ctx, req.GetId())
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.Wrapf(errox.NotFound, "report configuration with id '%s' does not exist", req.GetId())
	}
	if err := rejectNodeReportConfiguration(config); err != nil {
		return nil, err
	}
	// Remove report configs with empty scope. This can happen after downgrade to a version that has less scoping methods and doesn't support the new scoping method.
	if !common.HasValidResourceScope(config.GetResourceScope()) {
		return nil, errors.Wrapf(errox.InvalidArgs,
			"Report configuration '%s' has an empty resource scope (no collection ID or entity scope)", req.GetId())
	}

	converted, err := s.convertProtoReportConfigurationToV2(config)
	if err != nil {
		return nil, errors.Wrapf(err, "Error converting storage report configuration with id %s to response", config.GetId())
	}
	return converted, nil
}

func (s *serviceImpl) CountReportConfigurations(ctx context.Context, request *apiV2.RawQuery) (*apiV2.CountReportConfigurationsResponse, error) {
	parsedQuery, err := search.ParseQuery(request.GetQuery(), search.MatchAllIfEmpty())
	if err != nil {
		return nil, errors.Wrap(errox.InvalidArgs, err.Error())
	}
	parsedQuery = search.ConjunctionQuery(
		parsedQuery,
		search.NewQueryBuilder().AddExactMatches(search.ReportType, storage.ReportConfiguration_VULNERABILITY.String()).ProtoQuery(),
	)
	numReportConfigs, err := s.reportConfigStore.Count(ctx, parsedQuery)
	if err != nil {
		return nil, err
	}
	return &apiV2.CountReportConfigurationsResponse{Count: int32(numReportConfigs)}, nil
}

func (s *serviceImpl) DeleteReportConfiguration(ctx context.Context, id *apiV2.ResourceByID) (*apiV2.Empty, error) {
	if id.GetId() == "" {
		return nil, errors.Wrap(errox.InvalidArgs, "Report configuration id is required for deletion")
	}
	config, found, err := s.reportConfigStore.GetReportConfiguration(ctx, id.GetId())
	if err != nil {
		return nil, errors.Wrap(err, "Error finding report config")
	}
	if !found {
		return nil, errors.Wrapf(errox.NotFound, "Report config ID '%s' not found", id.GetId())
	}
	if err := rejectNodeReportConfiguration(config); err != nil {
		return nil, err
	}
	query := search.NewQueryBuilder().AddExactMatches(search.ReportConfigID, id.GetId()).AddExactMatches(search.ReportState, storage.ReportStatus_WAITING.String(), storage.ReportStatus_PREPARING.String()).ProtoQuery()
	reportSnapshots, _ := s.snapshotDatastore.SearchReportSnapshots(ctx, query)
	if len(reportSnapshots) > 0 {
		return &apiV2.Empty{}, errors.Wrapf(errox.InvalidArgs, "Report config ID '%s' has job in preparing or waiting state", id.GetId())
	}

	if err := s.reportConfigStore.RemoveReportConfiguration(ctx, id.GetId()); err != nil {
		return nil, err
	}

	if env.CentralWorkerEnabled.BooleanSetting() {
		notifyWithRetry(ctx, s.db, pgNotify.ReportConfigChanged, id.GetId())
	} else {
		s.scheduler.RemoveReportSchedule(id.GetId())
	}
	return &apiV2.Empty{}, nil
}

func (s *serviceImpl) GetReportStatus(ctx context.Context, req *apiV2.ResourceByID) (*apiV2.ReportStatusResponse, error) {
	if req == nil || req.GetId() == "" {
		return nil, errors.Wrap(errox.InvalidArgs, "Empty request or id")
	}
	rep, found, err := s.snapshotDatastore.Get(ctx, req.GetId())
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, errors.Wrapf(errox.NotFound, "Report snapshot not found for job id %s", req.GetId())
	}
	if err := rejectNodeReportSnapshot(rep); err != nil {
		return nil, err
	}
	status := s.convertPrototoV2Reportstatus(rep.GetReportStatus())
	return &apiV2.ReportStatusResponse{Status: status}, err
}

func (s *serviceImpl) GetReportHistory(ctx context.Context, req *apiV2.GetReportHistoryRequest) (*apiV2.ReportHistoryResponse, error) {
	if req == nil || req.GetId() == "" {
		return nil, errors.Wrap(errox.InvalidArgs, "Empty request or id")
	}
	parsedQuery, err := search.ParseQuery(req.GetReportParamQuery().GetQuery(), search.MatchAllIfEmpty())
	if err != nil {
		return nil, errors.Wrap(errox.InvalidArgs, err.Error())
	}

	conjunctionQuery := search.ConjunctionQuery(
		search.NewQueryBuilder().
			AddExactMatches(search.ReportConfigID, req.GetId()).
			AddExactMatches(search.ReportType, storage.ReportSnapshot_VULNERABILITY.String()).
			ProtoQuery(),
		parsedQuery,
	)
	// Fill in pagination.
	paginated.FillPaginationV2(conjunctionQuery, req.GetReportParamQuery().GetPagination(), maxPaginationLimit)

	results, err := s.snapshotDatastore.SearchReportSnapshots(ctx, conjunctionQuery)
	if err != nil {
		return nil, err
	}
	snapshots, err := s.convertProtoReportSnapshotstoV2(results)
	if err != nil {
		return nil, errors.Wrap(err, "Error converting storage report snapshots to response.")
	}
	res := apiV2.ReportHistoryResponse{
		ReportSnapshots: snapshots,
	}
	return &res, nil
}

func (s *serviceImpl) GetMyReportHistory(ctx context.Context, req *apiV2.GetReportHistoryRequest) (*apiV2.ReportHistoryResponse, error) {
	if req == nil || req.GetId() == "" {
		return nil, errors.Wrap(errox.InvalidArgs, "Empty request or id")
	}
	slimUser := authn.UserFromContext(ctx)
	if slimUser == nil {
		return nil, errors.New("Could not determine user identity from provided context")
	}

	parsedQuery, err := search.ParseQuery(req.GetReportParamQuery().GetQuery(), search.MatchAllIfEmpty())
	if err != nil {
		return nil, errors.Wrap(errox.InvalidArgs, err.Error())
	}

	err = verifyNoUserSearchLabels(parsedQuery)
	if err != nil {
		return nil, errors.Wrap(errox.InvalidArgs, err.Error())
	}

	conjunctionQuery := search.ConjunctionQuery(
		search.NewQueryBuilder().
			AddExactMatches(search.ReportConfigID, req.GetId()).
			AddExactMatches(search.UserID, slimUser.GetId()).
			AddExactMatches(search.ReportType, storage.ReportSnapshot_VULNERABILITY.String()).
			ProtoQuery(),
		parsedQuery,
	)

	// Fill in pagination.
	paginated.FillPaginationV2(conjunctionQuery, req.GetReportParamQuery().GetPagination(), maxPaginationLimit)

	results, err := s.snapshotDatastore.SearchReportSnapshots(ctx, conjunctionQuery)
	if err != nil {
		return nil, err
	}
	snapshots, err := s.convertProtoReportSnapshotstoV2(results)
	if err != nil {
		return nil, errors.Wrap(err, "Error converting storage report snapshots to response.")
	}
	res := apiV2.ReportHistoryResponse{
		ReportSnapshots: snapshots,
	}
	return &res, nil
}

func (s *serviceImpl) RunReport(ctx context.Context, req *apiV2.RunReportRequest) (*apiV2.RunReportResponse, error) {
	if err := sac.VerifyAuthzOK(workflowSAC.WriteAllowed(ctx)); err != nil {
		return nil, err
	}
	if req.GetReportConfigId() == "" {
		return nil, errors.Wrap(errox.InvalidArgs, "Report configuration ID is empty")
	}
	requesterID := authn.IdentityFromContextOrNil(ctx)
	if requesterID == nil {
		return nil, errors.New("Could not determine user identity from provided context")
	}

	var notificationMethod storage.ReportStatus_NotificationMethod
	if req.GetReportNotificationMethod() == apiV2.NotificationMethod_EMAIL {
		notificationMethod = storage.ReportStatus_EMAIL
	} else {
		notificationMethod = storage.ReportStatus_DOWNLOAD
	}

	reportReq, err := s.validator.ValidateAndGenerateReportRequest(req.GetReportConfigId(), notificationMethod,
		storage.ReportStatus_ON_DEMAND, requesterID)
	if err != nil {
		return nil, err
	}
	if reportReq.ReportSnapshot.GetType() == storage.ReportSnapshot_NODE_VULNERABILITY {
		return nil, errox.InvalidArgs.Newf("report configuration '%s' is a node vulnerability report; use the node report service", req.GetReportConfigId())
	}

	if env.CentralWorkerEnabled.BooleanSetting() {
		reportID, err := s.persistSnapshotAndNotify(ctx, reportReq, pgNotify.ReportRequestSubmitted)
		if err != nil {
			return nil, err
		}
		return &apiV2.RunReportResponse{
			ReportConfigId: req.GetReportConfigId(),
			ReportId:       reportID,
		}, nil
	}

	reportID, err := s.scheduler.SubmitReportRequest(ctx, reportReq, false)
	if err != nil {
		return nil, err
	}

	return &apiV2.RunReportResponse{
		ReportConfigId: req.GetReportConfigId(),
		ReportId:       reportID,
	}, nil
}

func (s *serviceImpl) CancelReport(ctx context.Context, req *apiV2.ResourceByID) (*apiV2.Empty, error) {
	if req.GetId() == "" {
		return nil, errors.Wrap(errox.InvalidArgs, "Report job ID is empty")
	}
	slimUser := authn.UserFromContext(ctx)
	if slimUser == nil {
		return nil, errors.New("Could not determine user identity from provided context")
	}

	err := s.validator.ValidateCancelReportRequest(req.GetId(), slimUser)
	if err != nil {
		return nil, err
	}

	if env.CentralWorkerEnabled.BooleanSetting() {
		notifyWithRetry(ctx, s.db, pgNotify.ReportRequestCancelled, req.GetId())
		return &apiV2.Empty{}, nil
	}

	cancelled, err := s.scheduler.CancelReportRequest(ctx, req.GetId())
	if err != nil {
		return nil, err
	}
	if !cancelled {
		return nil, errors.Wrapf(errox.InvariantViolation, "Cannot cancel. Report job ID '%s' no longer queued."+
			"It might already be preparing", req.GetId())
	}

	return &apiV2.Empty{}, nil
}

func (s *serviceImpl) DeleteReport(ctx context.Context, req *apiV2.DeleteReportRequest) (*apiV2.Empty, error) {
	if req == nil || req.GetId() == "" {
		return nil, errors.Wrap(errox.InvalidArgs, "Empty request or report job id")
	}

	slimUser := authn.UserFromContext(ctx)
	if slimUser == nil {
		return nil, errors.New("Could not determine user identity from provided context")
	}

	rep, found, err := s.snapshotDatastore.Get(ctx, req.GetId())
	if err != nil {
		return nil, errors.Wrapf(err, "Error finding report snapshot with job ID %q.", req.GetId())
	}

	if !found {
		return nil, errors.Wrapf(errox.NotFound, "Error finding report snapshot with job ID '%q'.", req.GetId())
	}
	if err := rejectNodeReportSnapshot(rep); err != nil {
		return nil, err
	}

	if slimUser.GetId() != rep.GetRequester().GetId() {
		return nil, errors.Wrap(errox.NotAuthorized, "Report cannot be deleted by a user who did not request the report.")
	}

	status := rep.GetReportStatus()
	if status.GetReportNotificationMethod() != storage.ReportStatus_DOWNLOAD {
		return nil, errors.Wrapf(errox.InvalidArgs, "Report job id %q did not generate a downloadable report and hence no report to delete.", req.GetId())
	}

	blobName := common.GetReportBlobPath(rep.GetReportConfigurationId(), req.GetId())
	switch status.GetRunState() {
	case storage.ReportStatus_FAILURE:
		return nil, errors.Wrapf(errox.InvalidArgs, "Report job %q has failed and no downloadable report to delete", req.GetId())
	case storage.ReportStatus_PREPARING, storage.ReportStatus_WAITING:
		return nil, errors.Wrapf(errox.InvalidArgs, "Report job %q is still running. Please cancel it or wait for its completion.", req.GetId())
	}

	ctx = sac.WithGlobalAccessScopeChecker(ctx,
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Administration)),
	)
	if err = s.blobStore.Delete(ctx, blobName); err != nil {
		return nil, errors.Wrapf(errox.InvariantViolation, "Failed to delete downloadable report %q", req.GetId())
	}
	return &apiV2.Empty{}, nil
}

// PostViewBasedReport validates a view-based report request and submits it to the report scheduler.
func (s *serviceImpl) PostViewBasedReport(ctx context.Context, req *apiV2.ReportRequestViewBased) (*apiV2.RunReportResponseViewBased, error) {
	// Check if view-based reports feature is enabled
	if !features.VulnerabilityViewBasedReports.Enabled() {
		return nil, errors.Wrap(errox.NotImplemented, "View-based vulnerability reports are not enabled. Please enable the ROX_VULNERABILITY_VIEW_BASED_REPORTS feature flag.")
	}

	if req == nil {
		return nil, errors.Wrap(errox.InvalidArgs, "Empty Request Body")
	}
	if req.GetType() == apiV2.ReportRequestViewBased_NODE_VULNERABILITY {
		return nil, errox.InvalidArgs.New("node vulnerability reports must be created via the node report service")
	}

	requesterID := authn.IdentityFromContextOrNil(ctx)
	if requesterID == nil {
		return nil, errors.New("Could not determine user identity from provided context")
	}

	// Validate the request and build the scheduler payload.
	reportReq, err := s.validator.ValidateAndGenerateViewBasedReportRequest(req, requesterID)
	if err != nil {
		return nil, err
	}

	if env.CentralWorkerEnabled.BooleanSetting() {
		reportID, err := s.persistSnapshotAndNotify(ctx, reportReq, pgNotify.ReportRequestSubmitted)
		if err != nil {
			return nil, err
		}
		return &apiV2.RunReportResponseViewBased{ReportID: reportID, RequestName: reportReq.ReportSnapshot.GetName()}, nil
	}

	reportID, err := s.scheduler.SubmitReportRequest(ctx, reportReq, false)
	if err != nil {
		return nil, errors.Wrapf(errox.ServerError, "Scheduler error:%s", err)
	}

	return &apiV2.RunReportResponseViewBased{ReportID: reportID, RequestName: reportReq.ReportSnapshot.GetName()}, nil
}

func (s *serviceImpl) GetViewBasedReportHistory(ctx context.Context, req *apiV2.GetViewBasedReportHistoryRequest) (*apiV2.ReportHistoryResponse, error) {
	// Check if view-based reports feature is enabled
	if !features.VulnerabilityViewBasedReports.Enabled() {
		return nil, errors.Wrap(errox.NotImplemented, "View-based vulnerability reports are not enabled. Please enable the ROX_VULNERABILITY_VIEW_BASED_REPORTS feature flag.")
	}

	parsedQuery, err := search.ParseQuery(req.GetReportParamQuery().GetQuery(), search.MatchAllIfEmpty())
	if err != nil {
		return nil, errors.Wrap(errox.InvalidArgs, err.Error())
	}

	conjunctionQuery := search.ConjunctionQuery(
		search.NewQueryBuilder().AddExactMatches(
			search.ReportRequestType,
			storage.ReportStatus_VIEW_BASED.String()).
			AddExactMatches(search.ReportType, storage.ReportSnapshot_VULNERABILITY.String()).
			ProtoQuery(),
		parsedQuery,
	)
	// Fill in pagination.
	paginated.FillPaginationV2(conjunctionQuery, req.GetReportParamQuery().GetPagination(), maxPaginationLimit)

	// View-based history endpoints are authorized with only Image+Deployment view.
	// The snapshot datastore requires WorkflowAdministration read, so elevate the
	// context to that single read scope instead of granting unrestricted access.
	results, err := s.snapshotDatastore.SearchReportSnapshots(SnapshotReadContext(ctx), conjunctionQuery)
	if err != nil {
		return nil, err
	}
	snapshots, err := s.convertViewBasedProtoReportSnapshotstoV2(results)
	if err != nil {
		return nil, errors.Wrap(err, "Error converting storage report snapshots to response.")
	}
	res := apiV2.ReportHistoryResponse{
		ReportSnapshots: snapshots,
	}
	return &res, nil
}

func (s *serviceImpl) GetViewBasedMyReportHistory(ctx context.Context, req *apiV2.GetViewBasedReportHistoryRequest) (*apiV2.ReportHistoryResponse, error) {
	// Check if view-based reports feature is enabled
	if !features.VulnerabilityViewBasedReports.Enabled() {
		return nil, errors.Wrap(errox.NotImplemented, "View-based vulnerability reports are not enabled. Please enable the ROX_VULNERABILITY_VIEW_BASED_REPORTS feature flag.")
	}

	slimUser := authn.UserFromContext(ctx)
	if slimUser == nil {
		return nil, errors.New("Could not determine user identity from provided context")
	}

	parsedQuery, err := search.ParseQuery(req.GetReportParamQuery().GetQuery(), search.MatchAllIfEmpty())
	if err != nil {
		return nil, errors.Wrap(errox.InvalidArgs, err.Error())
	}

	err = verifyNoUserSearchLabels(parsedQuery)
	if err != nil {
		return nil, errors.Wrap(errox.InvalidArgs, err.Error())
	}

	conjunctionQuery := search.ConjunctionQuery(
		search.NewQueryBuilder().
			AddExactMatches(search.UserID, slimUser.GetId()).
			AddExactMatches(search.ReportRequestType, storage.ReportStatus_VIEW_BASED.String()).
			AddExactMatches(search.ReportType, storage.ReportSnapshot_VULNERABILITY.String()).
			ProtoQuery(),
		parsedQuery,
	)

	// Fill in pagination.
	paginated.FillPaginationV2(conjunctionQuery, req.GetReportParamQuery().GetPagination(), maxPaginationLimit)

	// View-based history endpoints are authorized with only Image+Deployment view.
	// The snapshot datastore requires WorkflowAdministration read, so elevate the
	// context to that single read scope. Results are already scoped to the requesting
	// user via the UserID query filter above.
	results, err := s.snapshotDatastore.SearchReportSnapshots(SnapshotReadContext(ctx), conjunctionQuery)
	if err != nil {
		return nil, err
	}
	snapshots, err := s.convertViewBasedProtoReportSnapshotstoV2(results)
	if err != nil {
		return nil, errors.Wrap(err, "Error converting storage report snapshots to response.")
	}
	res := apiV2.ReportHistoryResponse{
		ReportSnapshots: snapshots,
	}
	return &res, nil
}

func verifyNoUserSearchLabels(q *v1.Query) error {
	return VerifyNoUserSearchLabels(q)
}

// VerifyNoUserSearchLabels rejects queries that filter on user identity fields.
func VerifyNoUserSearchLabels(q *v1.Query) error {
	unexpectedLabels := set.NewStringSet(search.UserID.String(), search.UserName.String())
	var err error
	search.ApplyFnToAllBaseQueries(q, func(bq *v1.BaseQuery) {
		mfQ, ok := bq.GetQuery().(*v1.BaseQuery_MatchFieldQuery)
		if ok && unexpectedLabels.Contains(mfQ.MatchFieldQuery.GetField()) {
			err = errors.New("query contains user search labels")
			return
		}
	})
	return err
}

func notifyWithRetry(ctx context.Context, db postgres.DB, channel, payload string) {
	NotifyWithRetry(ctx, db, channel, payload)
}

// NotifyWithRetry sends a pg_notify on the given channel, retrying a few times on failure.
// Failures are logged and not returned so that the originating RPC can still succeed.
func NotifyWithRetry(ctx context.Context, db postgres.DB, channel, payload string) {
	if db == nil {
		return
	}
	err := retry.WithRetry(func() error {
		return pgNotify.Notify(ctx, db, channel, payload)
	}, retry.Tries(3), retry.BetweenAttempts(func(previousAttempt int) {
		log.Errorf("pg_notify %s failed (attempt %d), retrying", channel, previousAttempt+1)
	}))
	if err != nil {
		log.Errorf("pg_notify %s failed after retries: %v", channel, err)
	}
}

func (s *serviceImpl) persistSnapshotAndNotify(ctx context.Context, reportReq *reportGen.ReportRequest, channel string) (string, error) {
	return PersistSnapshotAndNotify(ctx, s.validator, s.db, reportReq, channel)
}

func rejectNodeReportConfiguration(config *storage.ReportConfiguration) error {
	if config.GetType() == storage.ReportConfiguration_NODE_VULNERABILITY {
		return errox.InvalidArgs.Newf("report configuration '%s' is a node vulnerability report; use the node report service", config.GetId())
	}
	return nil
}

func rejectNodeReportSnapshot(snapshot *storage.ReportSnapshot) error {
	if snapshot.GetType() == storage.ReportSnapshot_NODE_VULNERABILITY {
		return errox.InvalidArgs.Newf("report job '%s' is a node vulnerability report; use the node report service", snapshot.GetReportId())
	}
	return nil
}

// SnapshotReadContext elevates the request context with WorkflowAdministration
// read so view-based history RPCs can query the snapshot datastore.
func SnapshotReadContext(ctx context.Context) context.Context {
	return sac.WithGlobalAccessScopeChecker(ctx,
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.WorkflowAdministration),
		),
	)
}

// PersistSnapshotAndNotify persists a WAITING snapshot and notifies the worker.
func PersistSnapshotAndNotify(ctx context.Context, validator *validation.Validator, db postgres.DB, reportReq *reportGen.ReportRequest, channel string) (string, error) {
	reportID, err := validator.PersistReportSnapshot(ctx, reportReq.ReportSnapshot)
	if err != nil {
		return "", err
	}
	NotifyWithRetry(ctx, db, channel, reportID)
	return reportID, nil
}
