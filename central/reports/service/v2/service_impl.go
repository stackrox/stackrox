package v2

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/pkg/errors"
	blobDS "github.com/stackrox/rox/central/blob/datastore"
	notifierDS "github.com/stackrox/rox/central/notifier/datastore"
	"github.com/stackrox/rox/central/reports/common"
	reportConfigDS "github.com/stackrox/rox/central/reports/config/datastore"
	schedulerV2 "github.com/stackrox/rox/central/reports/scheduler/v2"
	snapshotDS "github.com/stackrox/rox/central/reports/snapshot/datastore"
	"github.com/stackrox/rox/central/reports/validation"
	collectionDS "github.com/stackrox/rox/central/resourcecollection/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	apiV2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
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
		user.With(permissions.View(resources.WorkflowAdministration)): {
			"/v2.ReportService/ListReportConfigurations",
			"/v2.ReportService/GetReportConfiguration",
			"/v2.ReportService/CountReportConfigurations",
		},
		user.With(permissions.Modify(resources.WorkflowAdministration), permissions.View(resources.Integration)): {
			"/v2.ReportService/PostReportConfiguration",
			"/v2.ReportService/UpdateReportConfiguration",
		},
		user.With(permissions.Modify(resources.WorkflowAdministration)): {
			"/v2.ReportService/DeleteReportConfiguration",
		},
		user.With(permissions.View(resources.WorkflowAdministration)): {
			"/v2.ReportService/GetReportStatus",
			"/v2.ReportService/GetReportHistory",
			"/v2.ReportService/GetMyReportHistory",
		},
		user.With(permissions.Modify(resources.WorkflowAdministration)): {
			"/v2.ReportService/RunReport",
			"/v2.ReportService/CancelReport",
			"/v2.ReportService/DeleteReport",
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
}

func (s *serviceImpl) RegisterServiceServer(grpcServer *grpc.Server) {
	apiV2.RegisterReportServiceServer(grpcServer, s)
}

func (s *serviceImpl) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	if features.VulnReportingEnhancements.Enabled() {
		return apiV2.RegisterReportServiceHandler(ctx, mux, conn)
	}
	return nil
}

func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, authorizer.Authorized(ctx, fullMethodName)
}

func (s *serviceImpl) PostReportConfiguration(ctx context.Context, request *apiV2.ReportConfiguration) (*apiV2.ReportConfiguration, error) {
	creatorID := authn.IdentityFromContextOrNil(ctx)
	if creatorID == nil {
		return nil, errors.New("Could not determine user identity from provided context")
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

	err = s.scheduler.UpsertReportSchedule(createdReportConfig)
	if err != nil {
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

	updatedConfig := s.convertV2ReportConfigurationToProto(request, currentConfig.GetCreator(),
		currentConfig.GetVulnReportFilters().GetAccessScopeRules())

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

func (s *serviceImpl) ListReportConfigurations(ctx context.Context, query *apiV2.RawQuery) (*apiV2.ListReportConfigurationsResponse, error) {
	// Fill in Query.
	parsedQuery, err := search.ParseQuery(query.GetQuery(), search.MatchAllIfEmpty())
	if err != nil {
		return nil, errors.Wrap(errox.InvalidArgs, err.Error())
	}
	filteredQ := common.WithoutV1ReportConfigs(parsedQuery)

	// Fill in pagination.
	paginated.FillPaginationV2(filteredQ, query.GetPagination(), maxPaginationLimit)

	reportConfigs, err := s.reportConfigStore.GetReportConfigurations(ctx, filteredQ)
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
	if !common.IsV2ReportConfig(config) {
		return nil, errors.Wrap(errox.InvalidArgs, "report configuration does not belong to reporting version 2.0")
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
	filteredQ := common.WithoutV1ReportConfigs(parsedQuery)

	numReportConfigs, err := s.reportConfigStore.Count(ctx, filteredQ)
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
	if !common.IsV2ReportConfig(config) {
		return nil, errors.Wrap(errox.InvalidArgs, "report configuration does not belong to reporting version 2.0")
	}
	query := search.NewQueryBuilder().AddExactMatches(search.ReportConfigID, id.GetId()).AddExactMatches(search.ReportState, storage.ReportStatus_WAITING.String(), storage.ReportStatus_PREPARING.String()).ProtoQuery()
	reportSnapshots, _ := s.snapshotDatastore.SearchReportSnapshots(ctx, query)
	if len(reportSnapshots) > 0 {
		return &apiV2.Empty{}, errors.Wrapf(errox.InvalidArgs, "Report config ID '%s' has job in preparing or waiting state", id.GetId())
	}

	if err := s.reportConfigStore.RemoveReportConfiguration(ctx, id.GetId()); err != nil {
		return nil, err
	}

	s.scheduler.RemoveReportSchedule(id.GetId())
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
		search.NewQueryBuilder().AddExactMatches(search.ReportConfigID, req.GetId()).ProtoQuery(),
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
			AddExactMatches(search.UserID, slimUser.GetId()).ProtoQuery(),
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
	if err := sac.VerifyAuthzOK(workflowSAC.WriteAllowed(ctx)); err != nil {
		return nil, err
	}
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

func verifyNoUserSearchLabels(q *v1.Query) error {
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
