package v2

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/reportconfigurations/datastore"
	"github.com/stackrox/rox/central/reportconfigurations/service/common"
	"github.com/stackrox/rox/central/reports/manager"
	v1 "github.com/stackrox/rox/generated/api/v1"
	apiV2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/logging"
	reportConfigConverter "github.com/stackrox/rox/pkg/protoconv/reportconfigurations"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/paginated"
	"google.golang.org/grpc"
)

const maxPaginationLimit = 1000

var (
	log = logging.LoggerForModule()
)

type serviceImpl struct {
	apiV2.UnimplementedReportConfigurationServiceServer

	manager           manager.Manager
	reportConfigStore datastore.DataStore
	validator         *common.Validator
}

func (s *serviceImpl) RegisterServiceServer(grpcServer *grpc.Server) {
	if features.VulnMgmtReportingEnhancements.Enabled() {
		apiV2.RegisterReportConfigurationServiceServer(grpcServer, s)
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

func (s *serviceImpl) PostReportConfiguration(ctx context.Context, request *apiV2.ReportConfiguration) (*apiV2.ReportConfiguration, error) {
	protoReportConfig := reportConfigConverter.ConvertV2ReportConfigurationToProto(request)
	if err := s.validator.ValidateReportConfiguration(ctx, protoReportConfig); err != nil {
		return nil, errors.Errorf("Report config validation failed : %s", err)
	}
	id, err := s.reportConfigStore.AddReportConfiguration(ctx, protoReportConfig)
	if err != nil {
		return nil, err
	}

	createdReportConfig, _, err := s.reportConfigStore.GetReportConfiguration(ctx, id)
	if err != nil {
		return nil, err
	}
	// TODO ROX-16567 : Integrate with report manager when new reporting is implemented
	// if err := s.manager.Upsert(ctx, createdReportConfig); err != nil {
	//	 return nil, err
	// }

	return reportConfigConverter.ConvertProtoReportConfigurationToV2(createdReportConfig), nil
}

func (s *serviceImpl) UpdateReportConfiguration(ctx context.Context, request *apiV2.ReportConfiguration) (*v1.Empty, error) {
	protoReportConfig := reportConfigConverter.ConvertV2ReportConfigurationToProto(request)
	if err := s.validator.ValidateReportConfiguration(ctx, protoReportConfig); err != nil {
		return nil, errors.Errorf("Report config validation failed : %s", err)
	}

	// TODO ROX-16567 : Integrate with report manager when new reporting is implemented
	// if err := s.manager.Upsert(ctx, protoReportConfig); err != nil {
	//	return nil, err
	//}

	err := s.reportConfigStore.UpdateReportConfiguration(ctx, protoReportConfig)
	if err != nil {
		return nil, err
	}
	return &v1.Empty{}, nil
}

func (s *serviceImpl) GetReportConfigurations(ctx context.Context, query *v1.RawQuery) (*apiV2.GetReportConfigurationsResponse, error) {
	// Fill in Query.
	parsedQuery, err := search.ParseQuery(query.GetQuery(), search.MatchAllIfEmpty())
	if err != nil {
		return nil, errors.Wrap(errox.InvalidArgs, err.Error())
	}

	// Fill in pagination.
	paginated.FillPagination(parsedQuery, query.GetPagination(), maxPaginationLimit)

	reportConfigs, err := s.reportConfigStore.GetReportConfigurations(ctx, parsedQuery)
	if err != nil {
		return nil, errors.Wrap(err, "failed to retrieve report configurations")
	}
	converted := make([]*apiV2.ReportConfiguration, 0, len(reportConfigs))
	for _, config := range reportConfigs {
		converted = append(converted, reportConfigConverter.ConvertProtoReportConfigurationToV2(config))
	}
	return &apiV2.GetReportConfigurationsResponse{ReportConfigs: converted}, nil
}

func (s *serviceImpl) GetReportConfiguration(ctx context.Context, id *v1.ResourceByID) (*apiV2.ReportConfiguration, error) {
	if id.GetId() == "" {
		return nil, errors.Wrap(errox.InvalidArgs, "Report configuration id is required")
	}
	reportConfig, exists, err := s.reportConfigStore.GetReportConfiguration(ctx, id.GetId())
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.Wrapf(errox.NotFound, "report configuration with id '%s' does not exist", id)
	}
	return reportConfigConverter.ConvertProtoReportConfigurationToV2(reportConfig), nil
}

func (s *serviceImpl) CountReportConfigurations(ctx context.Context, request *v1.RawQuery) (*apiV2.CountReportConfigurationsResponse, error) {
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

func (s *serviceImpl) DeleteReportConfiguration(ctx context.Context, id *v1.ResourceByID) (*v1.Empty, error) {
	if id.GetId() == "" {
		return nil, errors.Wrap(errox.InvalidArgs, "Report configuration id is required for deletion")
	}
	if err := s.reportConfigStore.RemoveReportConfiguration(ctx, id.GetId()); err != nil {
		return &v1.Empty{}, err
	}

	// TODO ROX-16567 : Integrate with report manager when new reporting is implemented
	// return &v1.Empty{}, s.manager.Remove(ctx, id.GetId())
	return &v1.Empty{}, nil
}
