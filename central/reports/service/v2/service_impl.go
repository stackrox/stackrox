package v2

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/pkg/errors"
	metadataDS "github.com/stackrox/rox/central/reports/metadata/datastore"
	"github.com/stackrox/rox/central/role/resources"
	apiV2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/search"
	"google.golang.org/grpc"
)

var (
	log = logging.LoggerForModule()

	authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		user.With(permissions.View(resources.WorkflowAdministration)): {
			"/v2.ReportService/GetReportStatus",
			"/v2.ReportService/GetLastReportStatusConfigID",
		},
	})
)

type serviceImpl struct {
	apiV2.UnimplementedReportServiceServer
	metadataDatastore metadataDS.DataStore
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

func (s *serviceImpl) GetReportStatus(ctx context.Context, req *apiV2.ResourceByID) (*apiV2.ReportStatus, error) {
	if req == nil || req.GetId() == "" {
		return nil, errors.New("Empty request or id")
	}
	rep, found, err := s.metadataDatastore.Get(ctx, req.GetId())
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, errors.Errorf("Report not found for id %s", req.GetId())
	}
	status := convertPrototoV2Reportstatus(rep.GetReportStatus())
	return status, err

}

func (s *serviceImpl) GetLastReportStatusConfigID(ctx context.Context, req *apiV2.ResourceByID) (*apiV2.ReportStatus, error) {
	if req == nil || req.GetId() == "" {
		return nil, errors.New("Empty request or id")
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
		return nil, nil
	}
	status := convertPrototoV2Reportstatus(results[0].GetReportStatus())
	return status, err

}
