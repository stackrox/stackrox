package service

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
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
	"github.com/stackrox/rox/pkg/sac/resources"
	"google.golang.org/grpc"
)

var (
	authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		user.With(permissions.View(resources.Administration)): {
			"/v1.TelemetryService/GetTelemetryConfiguration",
		},
		user.With(permissions.Modify(resources.Administration)): {
			"/v1.TelemetryService/ConfigureTelemetry",
		},
		user.Authenticated(): {
			"/v1.TelemetryService/GetConfig",
		},
	})
)

type serviceImpl struct {
	v1.UnimplementedTelemetryServiceServer
}

func (s *serviceImpl) RegisterServiceServer(server *grpc.Server) {
	v1.RegisterTelemetryServiceServer(server, s)
}

func (s *serviceImpl) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return v1.RegisterTelemetryServiceHandler(ctx, mux, conn)
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, authorizer.Authorized(ctx, fullMethodName)
}

// GetTelemetryConfiguration used to tell whether periodic telemetry collection
// (previous implementation) was enabled. Returns false unconditionally.
// Deprecated: the previous implementation is not used for periodic collection.
func (s *serviceImpl) GetTelemetryConfiguration(_ context.Context, _ *v1.Empty) (*storage.TelemetryConfiguration, error) {
	return &storage.TelemetryConfiguration{
		Enabled: false,
	}, nil
}

// ConfigureTelemetry used to enable or disable periodic telemetry collection.
// Deprecated: the previous implementation is not used for periodic collection.
func (s *serviceImpl) ConfigureTelemetry(_ context.Context, _ *v1.ConfigureTelemetryRequest) (*storage.TelemetryConfiguration, error) {
	return &storage.TelemetryConfiguration{Enabled: false}, nil
}

func (s *serviceImpl) GetConfig(ctx context.Context, _ *v1.Empty) (*central.TelemetryConfig, error) {
	cfg := centralclient.GetConfig()
	if cfg == nil {
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
