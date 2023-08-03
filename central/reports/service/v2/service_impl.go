package v2

import (
	"bytes"
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/pkg/errors"
	blobDS "github.com/stackrox/rox/central/blob/datastore"
	notifierDS "github.com/stackrox/rox/central/notifier/datastore"
	reportConfigDS "github.com/stackrox/rox/central/reportconfigurations/datastore"
	"github.com/stackrox/rox/central/reports/common"
	schedulerV2 "github.com/stackrox/rox/central/reports/scheduler/v2"
	snapshotDS "github.com/stackrox/rox/central/reports/snapshot/datastore"
	"github.com/stackrox/rox/central/reports/validation"
	collectionDS "github.com/stackrox/rox/central/resourcecollection/datastore"
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
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
	"google.golang.org/grpc"
)

var (
	log = logging.LoggerForModule()

	workflowSAC = sac.ForResource(resources.WorkflowAdministration)

	authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		user.With(permissions.View(resources.WorkflowAdministration)): {
			"/v2.ReportService/GetReportStatus",
			"/v2.ReportService/GetLastReportStatusConfigID",
			"/v2.ReportService/GetReportHistory",
			"/v2.ReportService/DownloadReport",
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
	blobStore           blobDS.Datastore
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

func (s *serviceImpl) DownloadReport(ctx context.Context, req *apiV2.DownloadReportRequest) (*apiV2.DownloadReportResponse, error) {
	if req == nil || req.GetId() == "" {
		return nil, errors.Wrap(errox.InvalidArgs, "Empty request or report id")
	}
	rep, found, err := s.snapshotDatastore.Get(ctx, req.GetId())
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, errors.Wrapf(errox.NotFound, "Report not found for id %s", req.GetId())
	}
	status := rep.GetReportStatus()

	if status.GetReportNotificationMethod() != storage.ReportStatus_DOWNLOAD {
		return nil, errors.Wrapf(errox.NotFound, "Report download is not requested for job id %q", req.GetId())
	}

	if status.GetRunState() != storage.ReportStatus_SUCCESS {
		return nil, errors.Wrapf(errox.NotFound, "Report job %q is not ready for download", req.GetId())
	}

	buf := bytes.NewBuffer(nil)
	_, exists, err := s.blobStore.Get(ctx, common.GetReportBlobPath(req.GetId(), rep.GetReportConfigurationId()), buf)
	if err != nil {
		return nil, errors.Wrapf(errox.InvariantViolation, "Failed to fetch report data")
	}

	if !exists {
		// If the blob does not exist, return no reader.
		return nil, errors.Wrapf(errox.NotFound, "Report data not found")
	}

	return &apiV2.DownloadReportResponse{Data: buf.Bytes()}, nil
}
