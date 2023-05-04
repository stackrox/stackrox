package v2

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	notifierDataStore "github.com/stackrox/rox/central/notifier/datastore"
	"github.com/stackrox/rox/central/reportconfigurations/datastore"
	"github.com/stackrox/rox/central/reportconfigurations/service/common"
	"github.com/stackrox/rox/central/reports/manager"
	collectionDataStore "github.com/stackrox/rox/central/resourcecollection/datastore"
	accessScopeStore "github.com/stackrox/rox/central/role/datastore"
	apiV2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/pkg/logging"
	"google.golang.org/grpc"
)

var (
	log = logging.LoggerForModule()
)

type serviceImpl struct {
	apiV2.UnimplementedReportConfigurationServiceServer

	manager             manager.Manager
	reportConfigStore   datastore.DataStore
	notifierStore       notifierDataStore.DataStore
	accessScopeStore    accessScopeStore.DataStore
	collectionDatastore collectionDataStore.DataStore
}

func (s *serviceImpl) RegisterServiceServer(grpcServer *grpc.Server) {
	apiV2.RegisterReportConfigurationServiceServer(grpcServer, s)
}

func (s *serviceImpl) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return apiV2.RegisterReportConfigurationServiceHandler(ctx, mux, conn)
}

func (*serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, common.Authorizer.Authorized(ctx, fullMethodName)
}

func (s *serviceImpl) PostReportConfiguration(ctx context.Context, request *apiV2.ReportConfiguration) (*apiV2.ReportConfiguration, error) {
	//if err := s.validateReportConfiguration(ctx, request); err != nil {
	//	return nil, err
	//}
	//id, err := s.reportConfigStore.AddReportConfiguration(ctx, request)
	//if err != nil {
	//	return nil, err
	//}
	//
	//createdReportConfig, _, err := s.reportConfigStore.GetReportConfiguration(ctx, id)
	//if err := s.manager.Upsert(ctx, createdReportConfig); err != nil {
	//	return nil, err
	//}
	//
	//return &v1.PostReportConfigurationResponse{
	//	ReportConfig: createdReportConfig,
	//}, err
}
