package node

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/pkg/errors"
	blobDS "github.com/stackrox/rox/central/blob/datastore"
	notifierDS "github.com/stackrox/rox/central/notifier/datastore"
	"github.com/stackrox/rox/central/reports/common"
	reportConfigDS "github.com/stackrox/rox/central/reports/config/datastore"
	schedulerV2 "github.com/stackrox/rox/central/reports/scheduler/v2"
	reportsv2 "github.com/stackrox/rox/central/reports/service/v2"
	snapshotDS "github.com/stackrox/rox/central/reports/snapshot/datastore"
	"github.com/stackrox/rox/central/reports/validation"
	apiV2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"github.com/stackrox/rox/pkg/postgres"
	pgNotify "github.com/stackrox/rox/pkg/postgres/notify"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/paginated"
	"github.com/stackrox/rox/pkg/stringutils"
	"google.golang.org/grpc"
)

const (
	maxPaginationLimit = 1000
)

var (
	workflowSAC = sac.ForResource(resources.WorkflowAdministration)

	authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		user.With(permissions.View(resources.WorkflowAdministration), permissions.View(resources.Node), permissions.View(resources.Cluster)): {
			apiV2.NodeReportService_ListNodeReportConfigurations_FullMethodName,
			apiV2.NodeReportService_GetNodeReportConfiguration_FullMethodName,
			apiV2.NodeReportService_CountNodeReportConfigurations_FullMethodName,
			apiV2.NodeReportService_GetNodeReportHistory_FullMethodName,
			apiV2.NodeReportService_GetMyNodeReportHistory_FullMethodName,
		},
		user.With(permissions.Modify(resources.WorkflowAdministration), permissions.View(resources.Integration), permissions.View(resources.Node), permissions.View(resources.Cluster)): {
			apiV2.NodeReportService_PostNodeReportConfiguration_FullMethodName,
			apiV2.NodeReportService_UpdateNodeReportConfiguration_FullMethodName,
		},
		user.With(permissions.Modify(resources.WorkflowAdministration), permissions.View(resources.Node), permissions.View(resources.Cluster)): {
			apiV2.NodeReportService_DeleteNodeReportConfiguration_FullMethodName,
			apiV2.NodeReportService_RunNodeReport_FullMethodName,
		},
		// Cancel/Delete job match the image service: View is enough for the requester's own job.
		// Modify(WorkflowAdministration) is checked in-handler when cancelling another user's job.
		// Design table listed Modify(WorkflowAdministration) on those RPCs; image-parity won in review.
		// GetNodeReportStatus uses the same View(Node)+View(Cluster) set so view-based users can poll jobs.
		user.With(permissions.View(resources.Node), permissions.View(resources.Cluster)): {
			apiV2.NodeReportService_GetNodeReportStatus_FullMethodName,
			apiV2.NodeReportService_GetViewBasedNodeReportHistory_FullMethodName,
			apiV2.NodeReportService_GetViewBasedMyNodeReportHistory_FullMethodName,
			apiV2.NodeReportService_PostViewBasedNodeReport_FullMethodName,
			apiV2.NodeReportService_DeleteNodeReport_FullMethodName,
			apiV2.NodeReportService_CancelNodeReport_FullMethodName,
		},
	})
)

type serviceImpl struct {
	apiV2.UnimplementedNodeReportServiceServer
	reportConfigStore reportConfigDS.DataStore
	snapshotDatastore snapshotDS.DataStore
	notifierDatastore notifierDS.DataStore
	scheduler         schedulerV2.Scheduler
	blobStore         blobDS.Datastore
	validator         *validation.Validator
	db                postgres.DB
}

func (s *serviceImpl) RegisterServiceServer(grpcServer *grpc.Server) {
	apiV2.RegisterNodeReportServiceServer(grpcServer, s)
}

func (s *serviceImpl) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return apiV2.RegisterNodeReportServiceHandler(ctx, mux, conn)
}

func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, authorizer.Authorized(ctx, fullMethodName)
}

func (s *serviceImpl) PostNodeReportConfiguration(ctx context.Context, request *apiV2.ReportConfiguration) (*apiV2.ReportConfiguration, error) {
	creatorID, err := identityFromContext(ctx)
	if err != nil {
		return nil, err
	}

	if request.GetType() != apiV2.ReportConfiguration_NODE_VULNERABILITY {
		return nil, errox.InvalidArgs.CausedBy("report type must be NODE_VULNERABILITY")
	}

	if err := s.validator.ValidateReportConfiguration(request); err != nil {
		return nil, errors.Wrap(err, "validating report configuration")
	}

	creator := &storage.SlimUser{
		Id:   creatorID.UID(),
		Name: stringutils.FirstNonEmpty(creatorID.FullName(), creatorID.FriendlyName()),
	}

	protoReportConfig, err := s.convertV2ReportConfigurationToProto(request, creator, common.ExtractAccessScopeRules(creatorID))
	if err != nil {
		return nil, errors.Wrap(err, "converting report configuration")
	}

	id, err := s.reportConfigStore.AddReportConfiguration(ctx, protoReportConfig)
	if err != nil {
		return nil, err
	}

	createdReportConfig, exists, err := s.reportConfigStore.GetReportConfiguration(ctx, id)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errox.NotFound.CausedByf("report configuration with id '%s' was created but could not be read back", id)
	}

	if env.CentralWorkerEnabled.BooleanSetting() {
		reportsv2.NotifyWithRetry(ctx, s.db, pgNotify.ReportConfigChanged, id)
	} else if err := s.scheduler.UpsertReportSchedule(createdReportConfig); err != nil {
		return nil, err
	}

	resp, err := s.convertProtoReportConfigurationToV2(createdReportConfig)
	if err != nil {
		return nil, errors.Wrap(err, "report config created, but encountered error generating the response")
	}
	return resp, nil
}

func (s *serviceImpl) UpdateNodeReportConfiguration(ctx context.Context, request *apiV2.ReportConfiguration) (*apiV2.Empty, error) {
	if request.GetId() == "" {
		return nil, errox.InvalidArgs.CausedBy("report configuration id is required")
	}

	if request.GetType() != apiV2.ReportConfiguration_NODE_VULNERABILITY {
		return nil, errox.InvalidArgs.CausedBy("report type must be NODE_VULNERABILITY")
	}

	if err := s.validator.ValidateReportConfiguration(request); err != nil {
		return nil, errors.Wrap(err, "validating report configuration")
	}

	currentConfig, exists, err := s.reportConfigStore.GetReportConfiguration(ctx, request.GetId())
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errox.NotFound.CausedByf("report configuration with id '%s' does not exist", request.GetId())
	}

	if currentConfig.GetType() != storage.ReportConfiguration_NODE_VULNERABILITY {
		return nil, errox.InvalidArgs.CausedByf("report configuration '%s' is not a node vulnerability report", request.GetId())
	}

	query := search.NewQueryBuilder().AddExactMatches(search.ReportConfigID, request.GetId()).AddExactMatches(search.ReportState, storage.ReportStatus_WAITING.String(), storage.ReportStatus_PREPARING.String()).ProtoQuery()
	reportSnapshots, err := s.snapshotDatastore.SearchReportSnapshots(ctx, query)
	if err != nil {
		return nil, err
	}
	slimUser, err := userFromContext(ctx)
	if err != nil {
		return nil, err
	}
	for _, reportSnapshot := range reportSnapshots {
		if slimUser.GetId() == reportSnapshot.GetRequester().GetId() {
			return nil, errox.InvalidArgs.CausedBy("user has a report job running for this configuration")
		}
	}

	var accessScopeRules []*storage.SimpleAccessScope_Rules
	if filters := currentConfig.GetNodeVulnReportFilters(); filters != nil {
		accessScopeRules = filters.GetAccessScopeRules()
	}
	updatedConfig, err := s.convertV2ReportConfigurationToProto(request, currentConfig.GetCreator(), accessScopeRules)
	if err != nil {
		return nil, errors.Wrap(err, "converting report configuration")
	}

	err = s.reportConfigStore.UpdateReportConfiguration(ctx, updatedConfig)
	if err != nil {
		return nil, err
	}

	if env.CentralWorkerEnabled.BooleanSetting() {
		reportsv2.NotifyWithRetry(ctx, s.db, pgNotify.ReportConfigChanged, updatedConfig.GetId())
	} else if err := s.scheduler.UpsertReportSchedule(updatedConfig); err != nil {
		return nil, err
	}
	return &apiV2.Empty{}, nil
}

func (s *serviceImpl) ListNodeReportConfigurations(ctx context.Context, query *apiV2.RawQuery) (*apiV2.ListReportConfigurationsResponse, error) {
	parsedQuery, err := search.ParseQuery(query.GetQuery(), search.MatchAllIfEmpty())
	if err != nil {
		return nil, errox.InvalidArgs.CausedBy(err.Error())
	}

	// Filter for NODE_VULNERABILITY type
	parsedQuery = search.ConjunctionQuery(
		parsedQuery,
		search.NewQueryBuilder().AddExactMatches(search.ReportType, storage.ReportConfiguration_NODE_VULNERABILITY.String()).ProtoQuery(),
	)

	paginated.FillPaginationV2(parsedQuery, query.GetPagination(), maxPaginationLimit)

	reportConfigs, err := s.reportConfigStore.GetReportConfigurations(ctx, parsedQuery)
	if err != nil {
		return nil, errors.Wrap(err, "failed to retrieve report configurations")
	}
	v2Configs := make([]*apiV2.ReportConfiguration, 0, len(reportConfigs))

	for _, config := range reportConfigs {
		converted, err := s.convertProtoReportConfigurationToV2(config)
		if err != nil {
			return nil, errors.Wrapf(err, "error converting storage report configuration with id %s to response", config.GetId())
		}
		v2Configs = append(v2Configs, converted)
	}
	return &apiV2.ListReportConfigurationsResponse{ReportConfigs: v2Configs}, nil
}

func (s *serviceImpl) GetNodeReportConfiguration(ctx context.Context, req *apiV2.ResourceByID) (*apiV2.ReportConfiguration, error) {
	if req.GetId() == "" {
		return nil, errox.InvalidArgs.CausedBy("report configuration id is required")
	}
	config, exists, err := s.reportConfigStore.GetReportConfiguration(ctx, req.GetId())
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errox.NotFound.CausedByf("report configuration with id '%s' does not exist", req.GetId())
	}

	if config.GetType() != storage.ReportConfiguration_NODE_VULNERABILITY {
		return nil, errox.InvalidArgs.CausedByf("report configuration '%s' is not a node vulnerability report", req.GetId())
	}

	if !common.HasValidResourceScope(config.GetResourceScope()) {
		return nil, errox.InvalidArgs.CausedByf("report configuration '%s' has an empty resource scope (no entity scope)", req.GetId())
	}

	converted, err := s.convertProtoReportConfigurationToV2(config)
	if err != nil {
		return nil, errors.Wrapf(err, "error converting storage report configuration with id %s to response", config.GetId())
	}
	return converted, nil
}

func (s *serviceImpl) CountNodeReportConfigurations(ctx context.Context, request *apiV2.RawQuery) (*apiV2.CountReportConfigurationsResponse, error) {
	parsedQuery, err := search.ParseQuery(request.GetQuery(), search.MatchAllIfEmpty())
	if err != nil {
		return nil, errox.InvalidArgs.CausedBy(err.Error())
	}

	// Filter for NODE_VULNERABILITY type
	parsedQuery = search.ConjunctionQuery(
		parsedQuery,
		search.NewQueryBuilder().AddExactMatches(search.ReportType, storage.ReportConfiguration_NODE_VULNERABILITY.String()).ProtoQuery(),
	)

	numReportConfigs, err := s.reportConfigStore.Count(ctx, parsedQuery)
	if err != nil {
		return nil, err
	}
	return &apiV2.CountReportConfigurationsResponse{Count: int32(numReportConfigs)}, nil
}

func (s *serviceImpl) DeleteNodeReportConfiguration(ctx context.Context, id *apiV2.ResourceByID) (*apiV2.Empty, error) {
	if id.GetId() == "" {
		return nil, errox.InvalidArgs.CausedBy("report configuration id is required for deletion")
	}
	config, found, err := s.reportConfigStore.GetReportConfiguration(ctx, id.GetId())
	if err != nil {
		return nil, errors.Wrap(err, "error finding report config")
	}
	if !found {
		return nil, errox.NotFound.CausedByf("report config ID '%s' not found", id.GetId())
	}

	if config.GetType() != storage.ReportConfiguration_NODE_VULNERABILITY {
		return nil, errox.InvalidArgs.CausedByf("report configuration '%s' is not a node vulnerability report", id.GetId())
	}

	query := search.NewQueryBuilder().AddExactMatches(search.ReportConfigID, id.GetId()).AddExactMatches(search.ReportState, storage.ReportStatus_WAITING.String(), storage.ReportStatus_PREPARING.String()).ProtoQuery()
	reportSnapshots, err := s.snapshotDatastore.SearchReportSnapshots(ctx, query)
	if err != nil {
		return nil, errors.Wrap(err, "failed to search for active report snapshots")
	}
	if len(reportSnapshots) > 0 {
		return nil, errox.InvalidArgs.CausedByf("report config ID '%s' has job in preparing or waiting state", id.GetId())
	}

	if err := s.reportConfigStore.RemoveReportConfiguration(ctx, id.GetId()); err != nil {
		return nil, err
	}

	if env.CentralWorkerEnabled.BooleanSetting() {
		reportsv2.NotifyWithRetry(ctx, s.db, pgNotify.ReportConfigChanged, id.GetId())
	} else {
		s.scheduler.RemoveReportSchedule(id.GetId())
	}
	return &apiV2.Empty{}, nil
}

func (s *serviceImpl) RunNodeReport(ctx context.Context, req *apiV2.RunReportRequest) (*apiV2.RunReportResponse, error) {
	if req.GetReportConfigId() == "" {
		return nil, errox.InvalidArgs.CausedBy("report configuration id is required")
	}

	config, exists, err := s.reportConfigStore.GetReportConfiguration(ctx, req.GetReportConfigId())
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errox.NotFound.CausedByf("report configuration with id '%s' does not exist", req.GetReportConfigId())
	}

	if config.GetType() != storage.ReportConfiguration_NODE_VULNERABILITY {
		return nil, errox.InvalidArgs.CausedByf("report configuration '%s' is not a node vulnerability report", req.GetReportConfigId())
	}

	requesterID, err := identityFromContext(ctx)
	if err != nil {
		return nil, err
	}

	reportRequest, err := s.validator.ValidateAndGenerateReportRequest(
		req.GetReportConfigId(),
		storage.ReportStatus_NotificationMethod(req.GetReportNotificationMethod()),
		storage.ReportStatus_ON_DEMAND,
		requesterID,
	)
	if err != nil {
		return nil, errors.Wrap(err, "error validating report request")
	}

	if env.CentralWorkerEnabled.BooleanSetting() {
		reportID, err := reportsv2.PersistSnapshotAndNotify(ctx, s.validator, s.db, reportRequest, pgNotify.ReportRequestSubmitted)
		if err != nil {
			return nil, err
		}
		return &apiV2.RunReportResponse{ReportConfigId: req.GetReportConfigId(), ReportId: reportID}, nil
	}

	reportID, err := s.scheduler.SubmitReportRequest(ctx, reportRequest, false)
	if err != nil {
		return nil, errox.ServerError.CausedByf("scheduler error: %s", err)
	}

	return &apiV2.RunReportResponse{ReportConfigId: req.GetReportConfigId(), ReportId: reportID}, nil
}

// getReportHistory is a helper that fetches report snapshots with user filtering
func (s *serviceImpl) getReportHistory(ctx context.Context, queryBuilder *search.QueryBuilder, pagination *apiV2.Pagination, query string, userID string) (*apiV2.ReportHistoryResponse, error) {
	parsedQuery, err := search.ParseQuery(query, search.MatchAllIfEmpty())
	if err != nil {
		return nil, errox.InvalidArgs.CausedBy(err.Error())
	}

	baseQuery := queryBuilder.ProtoQuery()
	if userID != "" {
		if err := reportsv2.VerifyNoUserSearchLabels(parsedQuery); err != nil {
			return nil, errox.InvalidArgs.CausedBy(err.Error())
		}
		userIDQuery := search.NewQueryBuilder().AddExactMatches(search.UserID, userID).ProtoQuery()
		baseQuery = search.ConjunctionQuery(baseQuery, userIDQuery)
	}

	conjunctionQuery := search.ConjunctionQuery(baseQuery, parsedQuery)
	paginated.FillPaginationV2(conjunctionQuery, pagination, maxPaginationLimit)

	results, err := s.snapshotDatastore.SearchReportSnapshots(ctx, conjunctionQuery)
	if err != nil {
		return nil, err
	}

	blobNames, err := s.getExistingBlobNames(results)
	if err != nil {
		return nil, errors.Wrap(err, "failed to check blob availability")
	}

	v2Snapshots := make([]*apiV2.ReportSnapshot, 0, len(results))
	for _, snapshot := range results {
		converted, err := s.convertProtoReportSnapshotToV2(snapshot, blobNames)
		if err != nil {
			return nil, errors.Wrapf(err, "error converting snapshot with id %s", snapshot.GetReportId())
		}
		v2Snapshots = append(v2Snapshots, converted)
	}

	return &apiV2.ReportHistoryResponse{ReportSnapshots: v2Snapshots}, nil
}

func (s *serviceImpl) GetNodeReportHistory(ctx context.Context, req *apiV2.GetReportHistoryRequest) (*apiV2.ReportHistoryResponse, error) {
	if req == nil || req.GetId() == "" {
		return nil, errox.InvalidArgs.CausedBy("empty request or id")
	}

	queryBuilder := search.NewQueryBuilder().
		AddExactMatches(search.ReportConfigID, req.GetId()).
		AddExactMatches(search.ReportType, storage.ReportSnapshot_NODE_VULNERABILITY.String())
	return s.getReportHistory(ctx, queryBuilder, req.GetReportParamQuery().GetPagination(), req.GetReportParamQuery().GetQuery(), "")
}

func (s *serviceImpl) GetMyNodeReportHistory(ctx context.Context, req *apiV2.GetReportHistoryRequest) (*apiV2.ReportHistoryResponse, error) {
	if req == nil || req.GetId() == "" {
		return nil, errox.InvalidArgs.CausedBy("empty request or id")
	}

	slimUser, err := userFromContext(ctx)
	if err != nil {
		return nil, err
	}

	queryBuilder := search.NewQueryBuilder().
		AddExactMatches(search.ReportConfigID, req.GetId()).
		AddExactMatches(search.ReportType, storage.ReportSnapshot_NODE_VULNERABILITY.String())
	return s.getReportHistory(ctx, queryBuilder, req.GetReportParamQuery().GetPagination(), req.GetReportParamQuery().GetQuery(), slimUser.GetId())
}

func (s *serviceImpl) GetNodeReportStatus(ctx context.Context, req *apiV2.ResourceByID) (*apiV2.ReportStatusResponse, error) {
	if req == nil || req.GetId() == "" {
		return nil, errox.InvalidArgs.CausedBy("empty request or id")
	}
	rep, found, err := s.snapshotDatastore.Get(ctx, req.GetId())
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, errox.NotFound.CausedByf("report snapshot not found for job id %s", req.GetId())
	}
	if rep.GetType() != storage.ReportSnapshot_NODE_VULNERABILITY {
		return nil, errox.InvalidArgs.CausedByf("report snapshot '%s' is not a node vulnerability report", req.GetId())
	}
	status := convertPrototoV2Reportstatus(rep.GetReportStatus())
	return &apiV2.ReportStatusResponse{Status: status}, nil
}

func (s *serviceImpl) CancelNodeReport(ctx context.Context, req *apiV2.ResourceByID) (*apiV2.Empty, error) {
	if req == nil || req.GetId() == "" {
		return nil, errox.InvalidArgs.CausedBy("empty request or id")
	}
	snapshot, found, err := s.snapshotDatastore.Get(ctx, req.GetId())
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, errox.NotFound.CausedByf("report snapshot not found for id %s", req.GetId())
	}

	if snapshot.GetType() != storage.ReportSnapshot_NODE_VULNERABILITY {
		return nil, errox.InvalidArgs.CausedByf("report snapshot '%s' is not a node vulnerability report", req.GetId())
	}

	slimUser, err := userFromContext(ctx)
	if err != nil {
		return nil, err
	}
	if slimUser.GetId() != snapshot.GetRequester().GetId() {
		if err := sac.VerifyAuthzOK(workflowSAC.WriteAllowed(ctx)); err != nil {
			return nil, errors.Wrap(err, "user can only cancel a job created by the user unless they have Modify(WorkflowAdministration) permission")
		}
	}

	reportStatus := snapshot.GetReportStatus()
	if reportStatus.GetRunState() != storage.ReportStatus_WAITING && reportStatus.GetRunState() != storage.ReportStatus_PREPARING {
		return nil, errox.InvalidArgs.CausedBy("cannot cancel a job that is not in WAITING or PREPARING state")
	}

	if env.CentralWorkerEnabled.BooleanSetting() {
		reportsv2.NotifyWithRetry(ctx, s.db, pgNotify.ReportRequestCancelled, req.GetId())
		return &apiV2.Empty{}, nil
	}

	cancelled, err := s.scheduler.CancelReportRequest(ctx, req.GetId())
	if err != nil {
		return nil, err
	}
	if !cancelled {
		return nil, errors.Errorf("failed to cancel report job %s", req.GetId())
	}

	return &apiV2.Empty{}, nil
}

func (s *serviceImpl) DeleteNodeReport(ctx context.Context, req *apiV2.DeleteReportRequest) (*apiV2.Empty, error) {
	if req == nil || req.GetId() == "" {
		return nil, errox.InvalidArgs.CausedBy("empty request or id")
	}
	snapshot, found, err := s.snapshotDatastore.Get(ctx, req.GetId())
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, errox.NotFound.CausedByf("report snapshot not found for id %s", req.GetId())
	}

	if snapshot.GetType() != storage.ReportSnapshot_NODE_VULNERABILITY {
		return nil, errox.InvalidArgs.CausedByf("report snapshot '%s' is not a node vulnerability report", req.GetId())
	}

	slimUser, err := userFromContext(ctx)
	if err != nil {
		return nil, err
	}
	if slimUser.GetId() != snapshot.GetRequester().GetId() {
		return nil, errox.NotAuthorized.CausedBy("report cannot be deleted by a user who did not request the report")
	}

	status := snapshot.GetReportStatus()
	if status.GetReportNotificationMethod() != storage.ReportStatus_DOWNLOAD {
		return nil, errox.InvalidArgs.CausedByf("report job id %q did not generate a downloadable report and hence no report to delete", req.GetId())
	}
	switch status.GetRunState() {
	case storage.ReportStatus_FAILURE:
		return nil, errox.InvalidArgs.CausedByf("report job %q has failed and no downloadable report to delete", req.GetId())
	case storage.ReportStatus_PREPARING, storage.ReportStatus_WAITING:
		return nil, errox.InvalidArgs.CausedByf("report job %q is still running. Please cancel it or wait for its completion", req.GetId())
	}

	blobPath := common.GetReportBlobPath(reportBlobParentDir(snapshot), snapshot.GetReportId())
	blobCtx := sac.WithGlobalAccessScopeChecker(ctx,
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Administration),
		),
	)
	if err := s.blobStore.Delete(blobCtx, blobPath); err != nil {
		return nil, errox.InvariantViolation.CausedByf("failed to delete downloadable report %q", req.GetId())
	}

	return &apiV2.Empty{}, nil
}

func (s *serviceImpl) PostViewBasedNodeReport(ctx context.Context, req *apiV2.ReportRequestViewBased) (*apiV2.RunReportResponseViewBased, error) {
	if req.GetType() != apiV2.ReportRequestViewBased_NODE_VULNERABILITY {
		return nil, errox.InvalidArgs.CausedBy("report type must be NODE_VULNERABILITY")
	}

	requesterID, err := identityFromContext(ctx)
	if err != nil {
		return nil, err
	}

	reportRequest, err := s.validator.ValidateAndGenerateViewBasedReportRequest(req, requesterID)
	if err != nil {
		return nil, errors.Wrap(err, "error validating view-based report request")
	}

	if env.CentralWorkerEnabled.BooleanSetting() {
		reportID, err := reportsv2.PersistSnapshotAndNotify(ctx, s.validator, s.db, reportRequest, pgNotify.ReportRequestSubmitted)
		if err != nil {
			return nil, err
		}
		return &apiV2.RunReportResponseViewBased{ReportID: reportID, RequestName: reportRequest.ReportSnapshot.GetName()}, nil
	}

	reportID, err := s.scheduler.SubmitReportRequest(ctx, reportRequest, false)
	if err != nil {
		return nil, errox.ServerError.CausedByf("scheduler error: %s", err)
	}

	return &apiV2.RunReportResponseViewBased{ReportID: reportID, RequestName: reportRequest.ReportSnapshot.GetName()}, nil
}

func (s *serviceImpl) GetViewBasedNodeReportHistory(ctx context.Context, req *apiV2.GetViewBasedReportHistoryRequest) (*apiV2.ReportHistoryResponse, error) {
	queryBuilder := search.NewQueryBuilder().
		AddExactMatches(search.ReportType, storage.ReportSnapshot_NODE_VULNERABILITY.String()).
		AddExactMatches(search.ReportRequestType, storage.ReportStatus_VIEW_BASED.String())
	return s.getReportHistory(reportsv2.SnapshotReadContext(ctx), queryBuilder, req.GetReportParamQuery().GetPagination(), req.GetReportParamQuery().GetQuery(), "")
}

func (s *serviceImpl) GetViewBasedMyNodeReportHistory(ctx context.Context, req *apiV2.GetViewBasedReportHistoryRequest) (*apiV2.ReportHistoryResponse, error) {
	slimUser, err := userFromContext(ctx)
	if err != nil {
		return nil, err
	}

	queryBuilder := search.NewQueryBuilder().
		AddExactMatches(search.ReportType, storage.ReportSnapshot_NODE_VULNERABILITY.String()).
		AddExactMatches(search.ReportRequestType, storage.ReportStatus_VIEW_BASED.String())
	return s.getReportHistory(reportsv2.SnapshotReadContext(ctx), queryBuilder, req.GetReportParamQuery().GetPagination(), req.GetReportParamQuery().GetQuery(), slimUser.GetId())
}

func identityFromContext(ctx context.Context) (authn.Identity, error) {
	id := authn.IdentityFromContextOrNil(ctx)
	if id == nil {
		return nil, errox.NoCredentials.New("could not determine user identity from provided context")
	}
	return id, nil
}

func userFromContext(ctx context.Context) (*storage.SlimUser, error) {
	slimUser := authn.UserFromContext(ctx)
	if slimUser == nil {
		return nil, errox.NoCredentials.New("could not determine user identity from provided context")
	}
	return slimUser, nil
}

var _ Service = (*serviceImpl)(nil)
