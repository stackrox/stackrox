package service

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/integrationhealth/datastore"
	"github.com/stackrox/rox/central/role/resources"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"github.com/stackrox/rox/pkg/scanners"
	"google.golang.org/grpc"
)

var (
	authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		user.With(permissions.View(resources.ImageIntegration)): {
			"/v1.IntegrationHealthService/GetImageIntegrations",
		},
		user.With(permissions.View(resources.Notifier)): {
			"/v1.IntegrationHealthService/GetNotifiers",
		},
		user.With(permissions.View(resources.BackupPlugins)): {
			"/v1.IntegrationHealthService/GetBackupPlugins",
		},
		user.With(permissions.View(resources.ScannerDefinitions)): {
			"/v1.IntegrationHealthService/GetVulnDefinitionsInfo",
		},
	})
)

// ImageIntegrationService is the struct that manages the ImageIntegration API
type serviceImpl struct {
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
func (s *serviceImpl) GetImageIntegrations(ctx context.Context, empty *v1.Empty) (*v1.GetIntegrationHealthResponse, error) {
	healthData, err := s.datastore.GetRegistriesAndScanners(ctx)
	if err != nil {
		return nil, err
	}
	return &v1.GetIntegrationHealthResponse{
		IntegrationHealth: healthData,
	}, nil
}

// GetNotifiers returns the health status for all configured notifiers.
func (s *serviceImpl) GetNotifiers(ctx context.Context, empty *v1.Empty) (*v1.GetIntegrationHealthResponse, error) {
	healthData, err := s.datastore.GetNotifierPlugins(ctx)
	if err != nil {
		return nil, err
	}
	return &v1.GetIntegrationHealthResponse{
		IntegrationHealth: healthData,
	}, nil
}

// GetBackups returns the health status for all configured external backup integrations.
func (s *serviceImpl) GetBackupPlugins(ctx context.Context, empty *v1.Empty) (*v1.GetIntegrationHealthResponse, error) {
	healthData, err := s.datastore.GetBackupPlugins(ctx)
	if err != nil {
		return nil, err
	}
	return &v1.GetIntegrationHealthResponse{
		IntegrationHealth: healthData,
	}, nil
}

func (s *serviceImpl) GetVulnDefinitionsInfo(ctx context.Context, empty *v1.Empty) (*v1.VulnDefinitionsInfo, error) {
	info, err := s.vulnDefsInfoProvider.GetVulnDefsInfo()
	if err != nil {
		return nil, errors.Errorf("failed to obtain vulnerability definitions information: %v", err)
	}
	return info, nil
}
