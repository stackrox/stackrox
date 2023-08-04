package v2

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/pkg/errors"
	notifierDS "github.com/stackrox/rox/central/notifier/datastore"
	reportConfigDS "github.com/stackrox/rox/central/reports/config/datastore"
	schedulerV2 "github.com/stackrox/rox/central/reports/scheduler/v2"
	snapshotDS "github.com/stackrox/rox/central/reports/snapshot/datastore"
	"github.com/stackrox/rox/central/reports/validation"
	collectionDS "github.com/stackrox/rox/central/resourcecollection/datastore"
	apiV2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/paginated"
	"google.golang.org/grpc"
)

const (
	maxPaginationLimit = 1000
)

var (
	workflowSAC = sac.ForResource(resources.WorkflowAdministration)

	authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		user.With(permissions.View(resources.WorkflowAdministration)): {
			"/v2.ReportService/GetReportStatus",
			"/v2.ReportService/GetLastReportStatusConfigID",
			"/v2.ReportService/GetReportHistory",
		},
		user.With(permissions.Modify(resources.WorkflowAdministration)): {
			"/v2.ReportService/RunReport",
			"/v2.ReportService/CancelReport",
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
}

func (s *serviceImpl) RegisterServiceServer(grpcServer *grpc.Server) {
	apiV2.RegisterReportServiceServer(grpcServer, s)
}

func (s *serviceImpl) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	if env.VulnReportingEnhancements.BooleanSetting() {
		return apiV2.RegisterReportServiceHandler(ctx, mux, conn)
	}
	return nil
}

func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, authorizer.Authorized(ctx, fullMethodName)
}

func (s *serviceImpl) PostReportConfiguration(ctx context.Context, request *apiV2.ReportConfiguration) (*apiV2.ReportConfiguration, error) {
	slimUser := authn.UserFromContext(ctx)
	if slimUser == nil {
		return nil, errors.New("Could not determine user identity from provided context")
	}

	if err := s.ValidateReportConfiguration(request); err != nil {
		return nil, errors.Wrap(err, "Validating report configuration")
	}

	protoReportConfig := convertV2ReportConfigurationToProto(request)
	protoReportConfig.Creator = slimUser
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

	resp, err := convertProtoReportConfigurationToV2(createdReportConfig, s.collectionDatastore, s.notifierDatastore)
	if err != nil {
		return nil, errors.Wrap(err, "Report config created, but encountered error generating the response")
	}
	return resp, nil
}

func (s *serviceImpl) UpdateReportConfiguration(ctx context.Context, request *apiV2.ReportConfiguration) (*apiV2.Empty, error) {
	if request.GetId() == "" {
		return nil, errors.Wrap(errox.InvalidArgs, "Report configuration id is required")
	}
	if err := s.ValidateReportConfiguration(request); err != nil {
		return nil, errors.Wrap(err, "Validating report configuration")
	}

	protoReportConfig := convertV2ReportConfigurationToProto(request)

	err := s.reportConfigStore.UpdateReportConfiguration(ctx, protoReportConfig)
	if err != nil {
		return nil, err
	}

	err = s.scheduler.UpsertReportSchedule(protoReportConfig)
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

	// Fill in pagination.
	paginated.FillPaginationV2(parsedQuery, query.GetPagination(), maxPaginationLimit)

	reportConfigs, err := s.reportConfigStore.GetReportConfigurations(ctx, parsedQuery)
	if err != nil {
		return nil, errors.Wrap(err, "failed to retrieve report configurations")
	}
	v2Configs := make([]*apiV2.ReportConfiguration, 0, len(reportConfigs))

	for _, config := range reportConfigs {
		converted, err := convertProtoReportConfigurationToV2(config, s.collectionDatastore, s.notifierDatastore)
		if err != nil {
			return nil, errors.Wrapf(err, "Error converting storage report configuration with id %s to response", config.GetId())
		}
		v2Configs = append(v2Configs, converted)
	}
	return &apiV2.ListReportConfigurationsResponse{ReportConfigs: v2Configs}, nil
}

func (s *serviceImpl) GetReportConfiguration(ctx context.Context, id *apiV2.ResourceByID) (*apiV2.ReportConfiguration, error) {
	if id.GetId() == "" {
		return nil, errors.Wrap(errox.InvalidArgs, "Report configuration id is required")
	}
	config, exists, err := s.reportConfigStore.GetReportConfiguration(ctx, id.GetId())
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.Wrapf(errox.NotFound, "report configuration with id '%s' does not exist", id)
	}

	converted, err := convertProtoReportConfigurationToV2(config, s.collectionDatastore, s.notifierDatastore)
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
	status := convertPrototoV2Reportstatus(rep.GetReportStatus())
	return &apiV2.ReportStatusResponse{Status: status}, err
}

func (s *serviceImpl) GetLastReportStatusConfigID(ctx context.Context, req *apiV2.ResourceByID) (*apiV2.ReportStatusResponse, error) {
	if req == nil || req.GetId() == "" {
		return nil, errors.Wrap(errox.InvalidArgs, "Empty request or report config id")
	}
	query := search.NewQueryBuilder().AddExactMatches(search.ReportConfigID, req.GetId()).
		AddExactMatches(search.ReportState, storage.ReportStatus_SUCCESS.String(), storage.ReportStatus_FAILURE.String()).
		WithPagination(search.NewPagination().
			AddSortOption(search.NewSortOption(search.ReportCompletionTime).Reversed(true)).
			Limit(1)).ProtoQuery()
	results, err := s.snapshotDatastore.SearchReportSnapshots(ctx, query)
	if err != nil {
		return nil, err
	}
	if len(results) > 1 {
		return nil, errors.Errorf("Received %d records when only one record is expected", len(results))
	}
	if len(results) == 0 {
		return &apiV2.ReportStatusResponse{}, nil
	}
	status := convertPrototoV2Reportstatus(results[0].GetReportStatus())
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
	conjuncQuery := search.ConjunctionQuery(search.NewQueryBuilder().AddExactMatches(search.ReportConfigID, req.GetId()).ProtoQuery(), parsedQuery)
	results, err := s.snapshotDatastore.SearchReportSnapshots(ctx, conjuncQuery)
	if err != nil {
		return nil, err
	}
	snapshots := convertProtoReportSnapshotstoV2(results)
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
	slimUser := authn.UserFromContext(ctx)
	if slimUser == nil {
		return nil, errors.New("Could not determine user identity from provided context")
	}

	var notificationMethod storage.ReportStatus_NotificationMethod
	if req.GetReportNotificationMethod() == apiV2.NotificationMethod_EMAIL {
		notificationMethod = storage.ReportStatus_EMAIL
	} else {
		notificationMethod = storage.ReportStatus_DOWNLOAD
	}

	reportReq, err := validation.ValidateAndGenerateReportRequest(s.reportConfigStore, s.collectionDatastore, s.notifierDatastore,
		req.GetReportConfigId(), slimUser, notificationMethod, storage.ReportStatus_ON_DEMAND)
	if err != nil {
		return nil, err
	}

	reportID, err := s.scheduler.SubmitReportRequest(reportReq, false)
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

	err := validation.ValidateCancelReportRequest(s.snapshotDatastore, req.GetId(), slimUser)
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
