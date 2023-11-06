package service

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/complianceoperator/v2/compliancemanager"
	complianceDS "github.com/stackrox/rox/central/complianceoperator/v2/scanconfigurations/datastore"
	v2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/paginated"
	"google.golang.org/grpc"
)

const (
	maxPaginationLimit = 1000
)

var (
	authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		user.With(permissions.View(resources.Compliance)): {
			"/v2.ComplianceScanConfigurationService/ListComplianceScanConfigurations",
			"/v2.ComplianceScanConfigurationService/GetComplianceScanConfiguration",
			"/v2.ComplianceScanConfigurationService/GetComplianceScanConfigurationCount",
		},
		user.With(permissions.Modify(resources.Compliance)): {
			"/v2.ComplianceScanConfigurationService/CreateComplianceScanConfiguration",
			"/v2.ComplianceScanConfigurationService/DeleteComplianceScanConfiguration",
			"/v2.ComplianceScanConfigurationService/RunComplianceScanConfiguration",
			"/v2.ComplianceScanConfigurationService/UpdateComplianceScanConfiguration",
		},
	})
	log = logging.LoggerForModule()
)

// New returns a service object for registering with grpc.
func New(complianceScanSettingsDS complianceDS.DataStore, manager compliancemanager.Manager) Service {
	return &serviceImpl{
		complianceScanSettingsDS: complianceScanSettingsDS,
		manager:                  manager,
	}
}

type serviceImpl struct {
	v2.UnimplementedComplianceScanConfigurationServiceServer

	complianceScanSettingsDS complianceDS.DataStore
	manager                  compliancemanager.Manager
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *serviceImpl) RegisterServiceServer(grpcServer *grpc.Server) {
	v2.RegisterComplianceScanConfigurationServiceServer(grpcServer, s)
}

// RegisterServiceHandler registers this service with the given gRPC Gateway endpoint.
func (s *serviceImpl) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return v2.RegisterComplianceScanConfigurationServiceHandler(ctx, mux, conn)
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, authorizer.Authorized(ctx, fullMethodName)
}

func (s *serviceImpl) CreateComplianceScanConfiguration(ctx context.Context, req *v2.ComplianceScanConfiguration) (*v2.ComplianceScanConfiguration, error) {
	if req.GetScanName() == "" {
		return nil, errors.Wrap(errox.InvalidArgs, "Scan configuration name is required")
	}

	// Convert to storage type
	scanConfig := convertV2ScanConfigToStorage(ctx, req)

	// grab clusters
	var clusterIDs []string
	clusterIDs = append(clusterIDs, req.GetClusters()...)

	// Process scan request, config may be updated in the event of errors from sensor.
	scanConfig, err := s.manager.ProcessScanRequest(ctx, scanConfig, clusterIDs)
	if err != nil {
		return nil, errors.Wrapf(errox.InvalidArgs, "Unable to process scan config. %v", err)
	}

	return convertStorageScanConfigToV2(ctx, scanConfig, s.complianceScanSettingsDS)
}

func (s *serviceImpl) ListComplianceScanConfigurations(ctx context.Context, query *v2.RawQuery) (*v2.ListComplianceScanConfigurationsResponse, error) {
	// Fill in Query.
	parsedQuery, err := search.ParseQuery(query.GetQuery(), search.MatchAllIfEmpty())
	if err != nil {
		return nil, errors.Wrapf(errox.InvalidArgs, "Unable to parse query %v", err)
	}

	// Fill in pagination.
	paginated.FillPaginationV2(parsedQuery, query.GetPagination(), maxPaginationLimit)

	scanConfigs, err := s.complianceScanSettingsDS.GetScanConfigurations(ctx, parsedQuery)
	if err != nil {
		return nil, errors.Wrapf(errox.InvalidArgs, "Unable to retrieve scan configurations for query %v", query)
	}

	scanStatuses, err := convertStorageScanConfigToV2ScanStatuses(ctx, scanConfigs, s.complianceScanSettingsDS)
	if err != nil {
		return nil, errors.Wrap(errox.InvalidArgs, "failed to convert compliance scan configurations.")
	}

	return &v2.ListComplianceScanConfigurationsResponse{
		Configurations: scanStatuses,
	}, nil
}

func (s *serviceImpl) GetComplianceScanConfiguration(ctx context.Context, req *v2.ResourceByID) (*v2.ComplianceScanConfigurationStatus, error) {
	if req.GetId() == "" {
		return nil, errors.Wrap(errox.InvalidArgs, "Scan configuration name is required for retrieval")
	}

	scanConfig, found, err := s.complianceScanSettingsDS.GetScanConfiguration(ctx, req.GetId())
	if err != nil {
		return nil, errors.Wrapf(err, "failed to retrieve compliance scan configuration with id %q.", req.GetId())
	}
	if !found {
		return nil, errors.Errorf("failed to retrieve compliance scan configuration with id %q.", req.GetId())
	}

	return convertStorageScanConfigToV2ScanStatus(ctx, scanConfig, s.complianceScanSettingsDS)
}

func (s *serviceImpl) GetComplianceScanConfigurationCount(ctx context.Context, request *v2.RawQuery) (*v2.ComplianceScanConfigurationCount, error) {
	parsedQuery, err := search.ParseQuery(request.GetQuery(), search.MatchAllIfEmpty())
	if err != nil {
		return nil, errors.Wrap(errox.InvalidArgs, err.Error())
	}
	scanConfigs, err := s.complianceScanSettingsDS.CountScanConfigurations(ctx, parsedQuery)
	if err != nil {
		return nil, errors.Wrap(errox.NotFound, err.Error())
	}
	res := &v2.ComplianceScanConfigurationCount{
		Count: int32(scanConfigs),
	}
	return res, nil
}
