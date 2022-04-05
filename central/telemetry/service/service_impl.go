package service

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/stackrox/rox/central/telemetry/manager"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/auth/role/resources"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"google.golang.org/grpc"
)

var (
	authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		user.With(permissions.View(resources.DebugLogs)): {
			"/v1.TelemetryService/GetTelemetryConfiguration",
		},
		user.With(permissions.Modify(resources.DebugLogs)): {
			"/v1.TelemetryService/ConfigureTelemetry",
		},
	})
)

type serviceImpl struct {
	manager manager.Manager
}

func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, authorizer.Authorized(ctx, fullMethodName)
}

func (s *serviceImpl) RegisterServiceServer(server *grpc.Server) {
	v1.RegisterTelemetryServiceServer(server, s)
}

func (s *serviceImpl) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return v1.RegisterTelemetryServiceHandler(ctx, mux, conn)
}

func (s *serviceImpl) GetTelemetryConfiguration(ctx context.Context, _ *v1.Empty) (*storage.TelemetryConfiguration, error) {
	return s.manager.GetTelemetryConfig(ctx)
}

func (s *serviceImpl) ConfigureTelemetry(ctx context.Context, config *v1.ConfigureTelemetryRequest) (*storage.TelemetryConfiguration, error) {
	newConfig, err := s.manager.UpdateTelemetryConfig(ctx, config)
	if err != nil {
		return nil, err
	}
	return newConfig, nil
}
