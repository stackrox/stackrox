package v2

import (
	"context"
	"fmt"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/stackrox/rox/central/reportconfigurations/service/common"
	metadataDS "github.com/stackrox/rox/central/reports/metadata/datastore"
	apiV2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/logging"
	"google.golang.org/grpc"
)

const maxPaginationLimit = 1000

var (
	log = logging.LoggerForModule()
)

type serviceImpl struct {
	metadataDatastore metadataDS.DataStore
}

func (s *serviceImpl) RegisterServiceServer(grpcServer *grpc.Server) {
	if features.VulnMgmtReportingEnhancements.Enabled() {
		apiV2.RegisterReportServiceServer(grpcServer, s)
	}
}

func (s *serviceImpl) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	if features.VulnMgmtReportingEnhancements.Enabled() {
		return apiV2.RegisterReportConfigurationServiceHandler(ctx, mux, conn)
	}
	return nil
}

func (*serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, common.Authorizer.Authorized(ctx, fullMethodName)
}

func (s *serviceImpl) GetReportStatus(ctx context.Context, id *apiV2.ResourceByID) (*apiV2.ReportStatus, error) {
	rep, found, err := s.metadataDatastore.Get(ctx, id.GetId())
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, fmt.Errorf("Report not found for id %s", id.GetId())
	}
	status := convertPrototoV2Reportstatus(rep.GetReportStatus())
	return status, err

}
