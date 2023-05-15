package service

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/reportconfigurations/datastore"
	"github.com/stackrox/rox/central/reportconfigurations/service/common"
	"github.com/stackrox/rox/central/reports/manager"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/paginated"
	"google.golang.org/grpc"
)

var (
	log = logging.LoggerForModule()
)

type serviceImpl struct {
	v1.UnimplementedReportConfigurationServiceServer

	manager           manager.Manager
	reportConfigStore datastore.DataStore
	validator         *common.Validator
}

func (s *serviceImpl) GetReportConfigurations(ctx context.Context, query *v1.RawQuery) (*v1.GetReportConfigurationsResponse, error) {
	// Fill in Query.
	parsedQuery, err := search.ParseQuery(query.GetQuery(), search.MatchAllIfEmpty())
	if err != nil {
		return nil, errors.Wrap(errox.InvalidArgs, err.Error())
	}

	// Fill in pagination.
	paginated.FillPagination(parsedQuery, query.GetPagination(), 1000)

	reportConfigs, err := s.reportConfigStore.GetReportConfigurations(ctx, parsedQuery)
	if err != nil {
		return nil, errors.Wrap(err, "failed to retrieve report configurations")
	}
	return &v1.GetReportConfigurationsResponse{ReportConfigs: reportConfigs}, nil
}

func (s *serviceImpl) GetReportConfiguration(ctx context.Context, id *v1.ResourceByID) (*v1.GetReportConfigurationResponse, error) {
	reportConfig, exists, err := s.reportConfigStore.GetReportConfiguration(ctx, id.GetId())
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.Wrapf(errox.NotFound, "report configuration with id '%s' does not exist", id)
	}
	return &v1.GetReportConfigurationResponse{
		ReportConfig: reportConfig,
	}, nil
}

func (s *serviceImpl) PostReportConfiguration(ctx context.Context, request *v1.PostReportConfigurationRequest) (*v1.PostReportConfigurationResponse, error) {
	if err := s.validator.ValidateReportConfiguration(ctx, request.GetReportConfig()); err != nil {
		return nil, err
	}
	id, err := s.reportConfigStore.AddReportConfiguration(ctx, request.GetReportConfig())
	if err != nil {
		return nil, err
	}

	createdReportConfig, _, err := s.reportConfigStore.GetReportConfiguration(ctx, id)
	if err := s.manager.Upsert(ctx, createdReportConfig); err != nil {
		return nil, err
	}

	return &v1.PostReportConfigurationResponse{
		ReportConfig: createdReportConfig,
	}, err
}

func (s *serviceImpl) UpdateReportConfiguration(ctx context.Context, request *v1.UpdateReportConfigurationRequest) (*v1.Empty, error) {
	if err := s.validator.ValidateReportConfiguration(ctx, request.GetReportConfig()); err != nil {
		return &v1.Empty{}, err
	}
	if err := s.manager.Upsert(ctx, request.GetReportConfig()); err != nil {
		return nil, err
	}

	err := s.reportConfigStore.UpdateReportConfiguration(ctx, request.GetReportConfig())
	if err != nil {
		return nil, err
	}
	return &v1.Empty{}, nil
}

func (s *serviceImpl) DeleteReportConfiguration(ctx context.Context, id *v1.ResourceByID) (*v1.Empty, error) {
	if id.GetId() == "" {
		return nil, errors.Wrap(errox.InvalidArgs, "Report configuration id is required for deletion")
	}
	if err := s.reportConfigStore.RemoveReportConfiguration(ctx, id.GetId()); err != nil {
		return &v1.Empty{}, err
	}
	return &v1.Empty{}, s.manager.Remove(ctx, id.GetId())
}

func (s *serviceImpl) CountReportConfigurations(ctx context.Context, request *v1.RawQuery) (*v1.CountReportConfigurationsResponse, error) {
	parsedQuery, err := search.ParseQuery(request.GetQuery(), search.MatchAllIfEmpty())
	if err != nil {
		return nil, errors.Wrap(errox.InvalidArgs, err.Error())
	}

	numReportConfigs, err := s.reportConfigStore.Count(ctx, parsedQuery)
	if err != nil {
		return nil, err
	}
	return &v1.CountReportConfigurationsResponse{Count: int32(numReportConfigs)}, nil
}

func (s *serviceImpl) RegisterServiceServer(grpcServer *grpc.Server) {
	v1.RegisterReportConfigurationServiceServer(grpcServer, s)
}

func (s *serviceImpl) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return v1.RegisterReportConfigurationServiceHandler(ctx, mux, conn)
}

func (*serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, common.Authorizer.Authorized(ctx, fullMethodName)
}
