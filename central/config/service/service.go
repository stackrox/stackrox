package service

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/config/datastore"
	"github.com/stackrox/rox/central/telemetry/centralclient"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/features"
	pkgGRPC "github.com/stackrox/rox/pkg/grpc"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/allow"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sac/resources"
	"google.golang.org/grpc"
)

var (
	log        = logging.LoggerForModule()
	authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		allow.Anonymous(): {
			"/v1.ConfigService/GetPublicConfig",
		},
		user.With(permissions.View(resources.Administration)): {
			"/v1.ConfigService/GetConfig",
			"/v1.ConfigService/GetPrivateConfig",
			"/v1.ConfigService/GetVulnerabilityDeferralConfig",
		},
		user.With(permissions.Modify(resources.Administration)): {
			"/v1.ConfigService/PutConfig",
			"/v1.ConfigService/UpdateVulnerabilityDeferralConfig",
		},
	})
)

// Service provides the interface to modify Central config
type Service interface {
	pkgGRPC.APIService

	AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error)

	v1.ConfigServiceServer
}

// New returns a new Service instance using the given DataStore.
func New(datastore datastore.DataStore) Service {
	return &serviceImpl{
		datastore: datastore,
	}
}

type serviceImpl struct {
	v1.UnimplementedConfigServiceServer

	datastore datastore.DataStore
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *serviceImpl) RegisterServiceServer(grpcServer *grpc.Server) {
	v1.RegisterConfigServiceServer(grpcServer, s)
}

// RegisterServiceHandler registers this service with the given gRPC Gateway endpoint.
func (s *serviceImpl) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return v1.RegisterConfigServiceHandler(ctx, mux, conn)
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, authorizer.Authorized(ctx, fullMethodName)
}

// GetPublicConfig returns the publicly available config
func (s *serviceImpl) GetPublicConfig(_ context.Context, _ *v1.Empty) (*storage.PublicConfig, error) {
	publicConfig, err := s.datastore.GetPublicConfig()
	if err != nil {
		return nil, err
	}
	if publicConfig == nil {
		return &storage.PublicConfig{}, nil
	}
	return publicConfig, nil
}

// GetPrivateConfig returns the privately available config
func (s *serviceImpl) GetPrivateConfig(ctx context.Context, _ *v1.Empty) (*storage.PrivateConfig, error) {
	privateConfig, err := s.datastore.GetPrivateConfig(ctx)
	if err != nil {
		return nil, err
	}
	if privateConfig == nil {
		return &storage.PrivateConfig{}, nil
	}
	return privateConfig, nil
}

// GetConfig returns Central's config
func (s *serviceImpl) GetConfig(ctx context.Context, _ *v1.Empty) (*storage.Config, error) {
	config, err := s.datastore.GetConfig(ctx)
	if err != nil {
		return nil, err
	}
	if config == nil {
		return &storage.Config{}, nil
	}
	return config, nil
}

// PutConfig updates Central's config
func (s *serviceImpl) PutConfig(ctx context.Context, req *v1.PutConfigRequest) (*storage.Config, error) {
	if req.GetConfig() == nil {
		return nil, errors.Wrap(errox.InvalidArgs, "config must be specified")
	}
	if err := s.datastore.UpsertConfig(ctx, req.GetConfig()); err != nil {
		return nil, err
	}
	if req.GetConfig().GetPublicConfig().GetTelemetry().GetEnabled() {
		centralclient.Enable()
	} else {
		centralclient.Disable()
	}
	return req.GetConfig(), nil
}

// GetVulnerabilityDeferralConfig returns Central's vulnerability deferral configuration.
func (s *serviceImpl) GetVulnerabilityDeferralConfig(ctx context.Context, _ *v1.Empty) (*v1.GetVulnerabilityDeferralConfigResponse, error) {
	if !features.UnifiedCVEDeferral.Enabled() {
		return nil, errors.Errorf("Cannot fulfill request. Environment variable %s=false", features.UnifiedCVEDeferral.EnvVar())
	}
	privateConfig, err := s.datastore.GetPrivateConfig(ctx)
	if err != nil {
		return nil, err
	}
	return &v1.GetVulnerabilityDeferralConfigResponse{
		Config: VulnerabilityDeferralConfigStorageToV1(privateConfig.GetVulnerabilityDeferralConfig()),
	}, nil
}

// UpdateVulnerabilityDeferralConfig updates Central's vulnerability deferral configuration.
func (s *serviceImpl) UpdateVulnerabilityDeferralConfig(ctx context.Context, req *v1.UpdateVulnerabilityDeferralConfigRequest) (*v1.UpdateVulnerabilityDeferralConfigResponse, error) {
	if !features.UnifiedCVEDeferral.Enabled() {
		return nil, errors.Errorf("Cannot fulfill request. Environment variable %s=false", features.UnifiedCVEDeferral.EnvVar())
	}
	if req.GetConfig() == nil {
		return nil, errors.Wrap(errox.InvalidArgs, "vulnerability deferral config must be specified")
	}
	config, err := s.datastore.GetConfig(ctx)
	if err != nil {
		return nil, err
	}
	if config == nil {
		config = &storage.Config{}
	}
	if config.GetPrivateConfig() == nil {
		config.PrivateConfig = &storage.PrivateConfig{}
	}
	config.PrivateConfig.VulnerabilityDeferralConfig = VulnerabilityDeferralConfigV1ToStorage(req.GetConfig())
	if err := s.datastore.UpsertConfig(ctx, config); err != nil {
		return nil, err
	}

	return &v1.UpdateVulnerabilityDeferralConfigResponse{
		Config: req.GetConfig(),
	}, nil
}
