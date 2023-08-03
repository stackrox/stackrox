package v2

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/pkg/errors"
	notifierDS "github.com/stackrox/rox/central/notifier/datastore"
	"github.com/stackrox/rox/central/reportconfigurations/datastore"
	"github.com/stackrox/rox/central/reportconfigurations/service/common"
	schedulerV2 "github.com/stackrox/rox/central/reports/scheduler/v2"
	collectionDS "github.com/stackrox/rox/central/resourcecollection/datastore"
	apiV2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/logging"
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

	scheduler           schedulerV2.Scheduler
	reportConfigStore   datastore.DataStore
	collectionDatastore collectionDS.DataStore
	notifierDatastore   notifierDS.DataStore
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

	resp := convertProtoReportConfigurationToV2(createdReportConfig, s.collectionDatastore, s.notifierDatastore)
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
		converted := convertProtoReportConfigurationToV2(config, s.collectionDatastore, s.notifierDatastore)
		if converted != nil {
			v2Configs = append(v2Configs, converted)
		}
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

	converted := convertProtoReportConfigurationToV2(config, s.collectionDatastore, s.notifierDatastore)
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
