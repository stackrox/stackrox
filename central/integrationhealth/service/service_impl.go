package service

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/integrationhealth/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/scanners"
	"github.com/stackrox/rox/pkg/scanners/types"
	"google.golang.org/grpc"
)

var (
	authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		user.With(permissions.View(resources.Integration)): {
			v1.IntegrationHealthService_GetBackupPlugins_FullMethodName,
			v1.IntegrationHealthService_GetImageIntegrations_FullMethodName,
			v1.IntegrationHealthService_GetNotifiers_FullMethodName,
			v1.IntegrationHealthService_GetDeclarativeConfigs_FullMethodName,
		},
		user.With(permissions.View(resources.Administration)): {
			v1.IntegrationHealthService_GetVulnDefinitionsInfo_FullMethodName,
		},
	})
)

type serviceImpl struct {
	v1.UnimplementedIntegrationHealthServiceServer

	datastore            datastore.DataStore
	vulnDefsInfoProvider scanners.VulnDefsInfoProvider
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *serviceImpl) RegisterServiceServer(grpcServer *grpc.Server) {
	v1.RegisterIntegrationHealthServiceServer(grpcServer, s)
}

// RegisterServiceHandler registers this service with the given gRPC Gateway endpoint.
func (s *serviceImpl) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return v1.RegisterIntegrationHealthServiceHandler(ctx, mux, conn)
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, authorizer.Authorized(ctx, fullMethodName)
}

// GetImageIntegrations returns the health status for all configured registries and scanners.
func (s *serviceImpl) GetImageIntegrations(ctx context.Context, _ *v1.Empty) (*v1.GetIntegrationHealthResponse, error) {
	healthData, err := s.datastore.GetRegistriesAndScanners(ctx)
	if err != nil {
		return nil, err
	}
	return &v1.GetIntegrationHealthResponse{
		IntegrationHealth: healthData,
	}, nil
}

// GetNotifiers returns the health status for all configured notifiers.
func (s *serviceImpl) GetNotifiers(ctx context.Context, _ *v1.Empty) (*v1.GetIntegrationHealthResponse, error) {
	healthData, err := s.datastore.GetNotifierPlugins(ctx)
	if err != nil {
		return nil, err
	}
	return &v1.GetIntegrationHealthResponse{
		IntegrationHealth: healthData,
	}, nil
}

// GetBackupPlugins returns the health status for all configured external backup integrations.
func (s *serviceImpl) GetBackupPlugins(ctx context.Context, _ *v1.Empty) (*v1.GetIntegrationHealthResponse, error) {
	healthData, err := s.datastore.GetBackupPlugins(ctx)
	if err != nil {
		return nil, err
	}
	return &v1.GetIntegrationHealthResponse{
		IntegrationHealth: healthData,
	}, nil
}

// GetDeclarativeConfigs returns the health status for all declarative configurations.
func (s *serviceImpl) GetDeclarativeConfigs(ctx context.Context, _ *v1.Empty) (*v1.GetIntegrationHealthResponse, error) {
	healthData, err := s.datastore.GetDeclarativeConfigs(ctx)
	if err != nil {
		return nil, err
	}
	return &v1.GetIntegrationHealthResponse{IntegrationHealth: healthData}, nil
}

func (s *serviceImpl) GetVulnDefinitionsInfo(_ context.Context, req *v1.VulnDefinitionsInfoRequest) (*v1.VulnDefinitionsInfo, error) {
	scannerType := types.Clairify
	if req.GetComponent() == v1.VulnDefinitionsInfoRequest_SCANNER_V4 {
		scannerType = types.ScannerV4
	}

	info, err := s.vulnDefsInfoProvider.GetVulnDefsInfo(scannerType)
	if err != nil {
		return nil, errors.Errorf("failed to obtain vulnerability definitions information: %v", err)
	}
	return info, nil
}
