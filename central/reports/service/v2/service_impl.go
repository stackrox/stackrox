package v2

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/pkg/errors"
	reportConfigDS "github.com/stackrox/rox/central/reportconfigurations/datastore"
	metadataDS "github.com/stackrox/rox/central/reports/metadata/datastore"
	schedulerV2 "github.com/stackrox/rox/central/reports/scheduler/v2"
	reportGen "github.com/stackrox/rox/central/reports/scheduler/v2/reportgenerator"
	snapshotDS "github.com/stackrox/rox/central/reports/snapshot/datastore"
	apiV2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
	"google.golang.org/grpc"
)

var (
	log = logging.LoggerForModule()

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
	metadataDatastore metadataDS.DataStore
	reportConfigStore reportConfigDS.DataStore
	snapshotDatastore snapshotDS.DataStore
	scheduler         schedulerV2.Scheduler
}

func (s *serviceImpl) RegisterServiceServer(grpcServer *grpc.Server) {
	apiV2.RegisterReportServiceServer(grpcServer, s)

}

func (s *serviceImpl) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	if features.VulnMgmtReportingEnhancements.Enabled() {
		return apiV2.RegisterReportServiceHandler(ctx, mux, conn)
	}
	return nil
}

func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, authorizer.Authorized(ctx, fullMethodName)
}

func (s *serviceImpl) GetReportStatus(ctx context.Context, req *apiV2.ResourceByID) (*apiV2.ReportStatusResponse, error) {
	if req == nil || req.GetId() == "" {
		return nil, errors.Wrap(errox.InvalidArgs, "Empty request or id")
	}
	rep, found, err := s.metadataDatastore.Get(ctx, req.GetId())
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, errors.Wrapf(errox.NotFound, "Report not found for id %s", req.GetId())
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
	results, err := s.metadataDatastore.SearchReportMetadatas(ctx, query)
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
	if req == nil || req.GetReportConfigId() == "" {
		return nil, errors.Wrap(errox.InvalidArgs, "Empty request or id")
	}
	parsedQuery, err := search.ParseQuery(req.GetReportParamQuery().GetQuery(), search.MatchAllIfEmpty())
	if err != nil {
		return nil, errors.Wrap(errox.InvalidArgs, err.Error())
	}
	conjuncQuery := search.ConjunctionQuery(search.NewQueryBuilder().AddExactMatches(search.ReportConfigID, req.GetReportConfigId()).ProtoQuery(), parsedQuery)
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
	if req.GetReportConfigId() == "" {
		return nil, errors.Wrap(errox.InvalidArgs, "Report configuration ID is empty")
	}
	slimUser := authn.UserFromContext(ctx)
	if slimUser == nil {
		return nil, errors.New("Could not determine user identity from provided context")
	}
	config, found, err := s.reportConfigStore.GetReportConfiguration(ctx, req.GetReportConfigId())
	if err != nil {
		return nil, errors.Wrapf(err, "Error finding report configuration %s", req.GetReportConfigId())
	}
	if !found {
		return nil, errors.Wrapf(errox.NotFound, "Report configuration id not found %s", req.GetReportConfigId())
	}
	reportReq := &reportGen.ReportRequest{
		ReportConfig: config,
		ReportMetadata: &storage.ReportMetadata{
			ReportConfigId: req.GetReportConfigId(),
			Requester:      slimUser,
			ReportStatus: &storage.ReportStatus{
				RunState:          storage.ReportStatus_WAITING,
				ReportRequestType: storage.ReportStatus_ON_DEMAND,
			},
		},
	}
	if req.GetReportNotificationMethod() == apiV2.NotificationMethod_EMAIL {
		reportReq.ReportMetadata.ReportStatus.ReportNotificationMethod = storage.ReportStatus_EMAIL
	} else {
		reportReq.ReportMetadata.ReportStatus.ReportNotificationMethod = storage.ReportStatus_DOWNLOAD
		// Scope the downloadable reports to the access scope of user demanding the report
		reportReq.Ctx = ctx
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

func (s *serviceImpl) CancelReport(ctx context.Context, req *apiV2.ResourceByID) (*apiV2.CancelReportResponse, error) {
	if req.GetId() == "" {
		return nil, errors.Wrap(errox.InvalidArgs, "Report ID is empty")
	}
	cancelled, reason, err := s.scheduler.CancelReportRequest(ctx, req.GetId())
	if err != nil {
		return nil, err
	}
	return &apiV2.CancelReportResponse{
		Cancelled:      cancelled,
		FailureMessage: reason,
	}, nil
}
