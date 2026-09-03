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
	snapshotDS "github.com/stackrox/rox/central/reports/snapshot/datastore"
	"github.com/stackrox/rox/central/reports/validation"
	collectionDS "github.com/stackrox/rox/central/resourcecollection/datastore"
	apiV2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"github.com/stackrox/rox/pkg/logging"
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
	log = logging.LoggerForModule()

	workflowSAC = sac.ForResource(resources.WorkflowAdministration)

	authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		user.With(permissions.View(resources.WorkflowAdministration), permissions.View(resources.Node), permissions.View(resources.Cluster)): {
			apiV2.NodeReportService_ListNodeReportConfigurations_FullMethodName,
			apiV2.NodeReportService_GetNodeReportConfiguration_FullMethodName,
			apiV2.NodeReportService_CountNodeReportConfigurations_FullMethodName,
		},
		user.With(permissions.Modify(resources.WorkflowAdministration), permissions.View(resources.Integration), permissions.View(resources.Node), permissions.View(resources.Cluster)): {
			apiV2.NodeReportService_PostNodeReportConfiguration_FullMethodName,
			apiV2.NodeReportService_UpdateNodeReportConfiguration_FullMethodName,
		},
		user.With(permissions.Modify(resources.WorkflowAdministration), permissions.View(resources.Node), permissions.View(resources.Cluster)): {
			apiV2.NodeReportService_DeleteNodeReportConfiguration_FullMethodName,
		},
		user.With(permissions.Modify(resources.WorkflowAdministration), permissions.View(resources.Node), permissions.View(resources.Cluster)): {
			apiV2.NodeReportService_RunNodeReport_FullMethodName,
		},
		user.With(permissions.View(resources.WorkflowAdministration), permissions.View(resources.Node), permissions.View(resources.Cluster)): {
			apiV2.NodeReportService_GetNodeReportStatus_FullMethodName,
			apiV2.NodeReportService_GetNodeReportHistory_FullMethodName,
			apiV2.NodeReportService_GetMyNodeReportHistory_FullMethodName,
		},
		user.With(permissions.View(resources.Node), permissions.View(resources.Cluster)): {
			apiV2.NodeReportService_GetViewBasedNodeReportHistory_FullMethodName,
			apiV2.NodeReportService_GetViewBasedMyNodeReportHistory_FullMethodName,
			apiV2.NodeReportService_PostViewBasedNodeReport_FullMethodName,
		},
		user.With(permissions.View(resources.WorkflowAdministration), permissions.View(resources.Node), permissions.View(resources.Cluster)): {
			apiV2.NodeReportService_DeleteNodeReport_FullMethodName,
			apiV2.NodeReportService_CancelNodeReport_FullMethodName,
		},
	})
)

type serviceImpl struct {
	apiV2.UnimplementedNodeReportServiceServer
	reportConfigStore   reportConfigDS.DataStore
	snapshotDatastore   snapshotDS.DataStore
	collectionDatastore collectionDS.DataStore
	notifierDatastore   notifierDS.DataStore
	scheduler           schedulerV2.Scheduler
	blobStore           blobDS.Datastore
	validator           *validation.Validator
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
	creatorID := authn.IdentityFromContextOrNil(ctx)
	if creatorID == nil {
		return nil, errors.New("could not determine user identity from provided context")
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

	createdReportConfig, _, err := s.reportConfigStore.GetReportConfiguration(ctx, id)
	if err != nil {
		return nil, err
	}

	err = s.scheduler.UpsertReportSchedule(createdReportConfig)
	if err != nil {
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
	slimUser := authn.UserFromContext(ctx)
	if slimUser == nil {
		return nil, errors.New("could not determine user identity from provided context")
	}
	for _, reportSnapshot := range reportSnapshots {
		if slimUser.GetId() == reportSnapshot.GetRequester().GetId() {
			return nil, errox.InvalidArgs.CausedBy("user has a report job running for this configuration")
		}
	}

	updatedConfig, err := s.convertV2ReportConfigurationToProto(request, currentConfig.GetCreator(),
		currentConfig.GetNodeVulnReportFilters().GetAccessScopeRules())
	if err != nil {
		return nil, errors.Wrap(err, "converting report configuration")
	}

	err = s.reportConfigStore.UpdateReportConfiguration(ctx, updatedConfig)
	if err != nil {
		return nil, err
	}

	err = s.scheduler.UpsertReportSchedule(updatedConfig)
	if err != nil {
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
		return &apiV2.Empty{}, errox.InvalidArgs.CausedByf("report config ID '%s' has job in preparing or waiting state", id.GetId())
	}

	if err := s.reportConfigStore.RemoveReportConfiguration(ctx, id.GetId()); err != nil {
		return nil, err
	}

	s.scheduler.RemoveReportSchedule(id.GetId())
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

	reportRequest, err := s.validator.ValidateAndGenerateReportRequest(
		req.GetReportConfigId(),
		storage.ReportStatus_NotificationMethod(req.GetReportNotificationMethod()),
		storage.ReportStatus_ON_DEMAND,
		authn.IdentityFromContextOrNil(ctx),
	)
	if err != nil {
		return nil, errors.Wrap(err, "error validating report request")
	}

	// Submit to scheduler. on-demand reports are not re-submissions.
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
		userIDQuery := search.NewQueryBuilder().AddExactMatches(search.RequesterUserID, userID).ProtoQuery()
		baseQuery = search.ConjunctionQuery(baseQuery, userIDQuery)
	}

	conjunctionQuery := search.ConjunctionQuery(baseQuery, parsedQuery)
	paginated.FillPaginationV2(conjunctionQuery, pagination, maxPaginationLimit)

	results, err := s.snapshotDatastore.SearchReportSnapshots(ctx, conjunctionQuery)
	if err != nil {
		return nil, err
	}

	blobNames, err := s.getExistingBlobNames(ctx, results)
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

	slimUser := authn.UserFromContext(ctx)
	if slimUser == nil {
		return nil, errors.New("could not determine user identity from provided context")
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
	status := convertPrototoV2Reportstatus(rep.GetReportStatus())
	return &apiV2.ReportStatusResponse{Status: status}, err
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

	slimUser := authn.UserFromContext(ctx)
	if slimUser == nil {
		return nil, errors.New("could not determine user identity from provided context")
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

	slimUser := authn.UserFromContext(ctx)
	if slimUser == nil {
		return nil, errors.New("could not determine user identity from provided context")
	}
	if slimUser.GetId() != snapshot.GetRequester().GetId() {
		if err := sac.VerifyAuthzOK(workflowSAC.WriteAllowed(ctx)); err != nil {
			return nil, errors.Wrap(err, "user can only delete a job created by the user unless they have Modify(WorkflowAdministration) permission")
		}
	}

	if err := s.snapshotDatastore.DeleteReportSnapshot(ctx, req.GetId()); err != nil {
		return nil, err
	}

	// Delete associated blob if it exists
	parentDir := snapshot.GetReportConfigurationId()
	if snapshot.GetReportStatus().GetReportRequestType() == storage.ReportStatus_VIEW_BASED {
		parentDir = "view-based-report"
	}
	blobPath := common.GetReportBlobPath(parentDir, snapshot.GetReportId())
	if err := s.blobStore.Delete(ctx, blobPath); err != nil {
		log.Errorf("Error deleting blob %s: %v", blobPath, err)
	}

	return &apiV2.Empty{}, nil
}

func (s *serviceImpl) PostViewBasedNodeReport(ctx context.Context, req *apiV2.ReportRequestViewBased) (*apiV2.RunReportResponseViewBased, error) {
	if req.GetType() != apiV2.ReportRequestViewBased_NODE_VULNERABILITY {
		return nil, errox.InvalidArgs.CausedBy("report type must be NODE_VULNERABILITY")
	}

	requesterID := authn.IdentityFromContextOrNil(ctx)
	if requesterID == nil {
		return nil, errors.New("could not determine user identity from provided context")
	}

	reportRequest, err := s.validator.ValidateAndGenerateViewBasedReportRequest(req, requesterID)
	if err != nil {
		return nil, errors.Wrap(err, "error validating view-based report request")
	}

	// Submit to scheduler. view-based reports are always on-demand, not re-submissions.
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
	return s.getReportHistory(ctx, queryBuilder, req.GetReportParamQuery().GetPagination(), req.GetReportParamQuery().GetQuery(), "")
}

func (s *serviceImpl) GetViewBasedMyNodeReportHistory(ctx context.Context, req *apiV2.GetViewBasedReportHistoryRequest) (*apiV2.ReportHistoryResponse, error) {
	slimUser := authn.UserFromContext(ctx)
	if slimUser == nil {
		return nil, errors.New("could not determine user identity from provided context")
	}

	queryBuilder := search.NewQueryBuilder().
		AddExactMatches(search.ReportType, storage.ReportSnapshot_NODE_VULNERABILITY.String()).
		AddExactMatches(search.ReportRequestType, storage.ReportStatus_VIEW_BASED.String())
	return s.getReportHistory(ctx, queryBuilder, req.GetReportParamQuery().GetPagination(), req.GetReportParamQuery().GetQuery(), slimUser.GetId())
}

var _ Service = (*serviceImpl)(nil)
