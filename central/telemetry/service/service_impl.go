package service

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/stackrox/stackrox/central/role/resources"
	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/auth/permissions"
	"github.com/stackrox/stackrox/pkg/grpc/authz"
	"github.com/stackrox/stackrox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/stackrox/pkg/grpc/authz/user"
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

type serviceImpl struct{}

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
	return &storage.TelemetryConfiguration{
		Enabled: false,
	}, nil
}

func (s *serviceImpl) ConfigureTelemetry(ctx context.Context, config *v1.ConfigureTelemetryRequest) (*storage.TelemetryConfiguration, error) {
	return &storage.TelemetryConfiguration{Enabled: false}, nil
}
