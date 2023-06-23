package v2

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/reportconfigurations/service/common"
	metadataDS "github.com/stackrox/rox/central/reports/metadata/datastore"
	apiV2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/search"
	"google.golang.org/grpc"
)

var (
	log = logging.LoggerForModule()
)

type serviceImpl struct {
	apiV2.UnimplementedReportServiceServer
	metadataDatastore metadataDS.DataStore
}

func (s *serviceImpl) RegisterServiceServer(grpcServer *grpc.Server) {
	if features.VulnMgmtReportingEnhancements.Enabled() {
		apiV2.RegisterReportServiceServer(grpcServer, s)
	}
}

func (s *serviceImpl) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	if features.VulnMgmtReportingEnhancements.Enabled() {
		return apiV2.RegisterReportServiceHandler(ctx, mux, conn)
	}
	return nil
}

func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, common.Authorizer.Authorized(ctx, fullMethodName)
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

func (s *serviceImpl) GetReportStatusConfigID(ctx context.Context, req *apiV2.ResourceByID) (*apiV2.ReportStatus, error) {
	if req == nil || req.GetId() == "" {
		return nil, errors.New("Empty request or id")
	}
	result, err := s.metadataDatastore.SearchReportMetadatas(ctx, search.MatchFieldQuery(search.ReportConfigID.String(), req.GetId(), false))
	if err != nil {
		return nil, err
	}

	status := convertPrototoV2Reportstatus(result[0].GetReportStatus())
	return status, err

}
