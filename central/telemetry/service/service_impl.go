package service

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/central/telemetry/centralclient"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"google.golang.org/grpc"
)

var (
	authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		// TODO: ROX-12750 Replace DebugLogs with Administration.
		user.With(permissions.View(resources.DebugLogs)): {
			"/v1.TelemetryService/GetTelemetryConfiguration",
		},
		// TODO: ROX-12750 Replace DebugLogs with Administration.
		user.With(permissions.Modify(resources.DebugLogs)): {
			"/v1.TelemetryService/ConfigureTelemetry",
		},
		anyAuthenticated{}: {
			"/v1.TelemetryService/GetConfig",
		},
	})
)

type anyAuthenticated struct{}

// Authorized implements authz.Authorizer for anyAuthenticated struct.
func (anyAuthenticated) Authorized(ctx context.Context, fullMethodName string) error {
	id, err := authn.IdentityFromContext(ctx)
	if err != nil {
		return errox.NotAuthorized.CausedBy(err)
	}
	if id == nil || id.UID() == "" {
		return errox.NotAuthorized.CausedBy(errox.NoCredentials)
	}
	return nil
}

type serviceImpl struct {
	v1.UnimplementedTelemetryServiceServer
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
	return &storage.TelemetryConfiguration{
		Enabled: false,
	}, nil
}

func (s *serviceImpl) ConfigureTelemetry(ctx context.Context, config *v1.ConfigureTelemetryRequest) (*storage.TelemetryConfiguration, error) {
	return &storage.TelemetryConfiguration{Enabled: false}, nil
}

func (s *serviceImpl) GetConfig(ctx context.Context, _ *v1.Empty) (*central.TelemetryConfig, error) {
	cfg := centralclient.InstanceConfig()
	if !cfg.Enabled() {
		return nil, errox.NotFound.New("telemetry collection is disabled")
	}
	id, err := authn.IdentityFromContext(ctx)
	if err != nil {
		return nil, err
	}
	return &central.TelemetryConfig{
		UserId:       cfg.HashUserAuthID(id),
		Endpoint:     cfg.Endpoint,
		StorageKeyV1: cfg.StorageKey,
	}, nil
}
