package service

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/pkg/errors"
	notifierDS "github.com/stackrox/rox/central/notifier/datastore"
	"github.com/stackrox/rox/central/reports/common"
	"github.com/stackrox/rox/central/reports/config/datastore"
	"github.com/stackrox/rox/central/reports/manager"
	collectionDS "github.com/stackrox/rox/central/resourcecollection/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/paginated"
	"google.golang.org/grpc"
)

var (
	// authorizer is used for authorizing report configuration grpc service calls.
	authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		user.With(permissions.View(resources.WorkflowAdministration)): {
			"/v1.ReportConfigurationService/GetReportConfigurations",
			"/v1.ReportConfigurationService/GetReportConfiguration",
			"/v1.ReportConfigurationService/CountReportConfigurations",
		},
		user.With(permissions.Modify(resources.WorkflowAdministration), permissions.View(resources.Integration)): {
			"/v1.ReportConfigurationService/PostReportConfiguration",
			"/v1.ReportConfigurationService/UpdateReportConfiguration",
		},
		user.With(permissions.Modify(resources.WorkflowAdministration)): {
			"/v1.ReportConfigurationService/DeleteReportConfiguration",
		},
	})
)

type serviceImpl struct {
	v1.UnimplementedReportConfigurationServiceServer

	manager             manager.Manager
	reportConfigStore   datastore.DataStore
	collectionDatastore collectionDS.DataStore
	notifierDatastore   notifierDS.DataStore
}

func (s *serviceImpl) GetReportConfigurations(ctx context.Context, query *v1.RawQuery) (*v1.GetReportConfigurationsResponse, error) {
	// Fill in Query.
	parsedQuery, err := search.ParseQuery(query.GetQuery(), search.MatchAllIfEmpty())
	if err != nil {
		return nil, errors.Wrap(errox.InvalidArgs, err.Error())
	}

	filteredQ := common.WithoutV2ReportConfigs(parsedQuery)

	// Fill in pagination.
	paginated.FillPagination(filteredQ, query.GetPagination(), 1000)

	reportConfigs, err := s.reportConfigStore.GetReportConfigurations(ctx, filteredQ)
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
	if !common.IsV1ReportConfig(reportConfig) {
		return nil, errors.Wrap(errox.InvalidArgs, "report configuration does not belong to reporting version 1.0")
	}
	return &v1.GetReportConfigurationResponse{
		ReportConfig: reportConfig,
	}, nil
}

func (s *serviceImpl) PostReportConfiguration(ctx context.Context, request *v1.PostReportConfigurationRequest) (*v1.PostReportConfigurationResponse, error) {
	if err := s.validateReportConfiguration(ctx, request.GetReportConfig()); err != nil {
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
	if err := s.validateReportConfiguration(ctx, request.GetReportConfig()); err != nil {
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

	config, found, err := s.reportConfigStore.GetReportConfiguration(ctx, id.GetId())
	if err != nil {
		return nil, errors.Wrap(err, "Error finding report config")
	}
	if !found {
		return nil, errors.Wrapf(errox.NotFound, "Report config ID '%s' not found", id.GetId())
	}
	if !common.IsV1ReportConfig(config) {
		return nil, errors.Wrap(errox.InvalidArgs, "report configuration does not belong to reporting version 1.0")
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
	filteredQ := common.WithoutV2ReportConfigs(parsedQuery)

	numReportConfigs, err := s.reportConfigStore.Count(ctx, filteredQ)
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
	return ctx, authorizer.Authorized(ctx, fullMethodName)
}
