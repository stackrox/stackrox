package service

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/config/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/auth/role/resources"
	"github.com/stackrox/rox/pkg/errorhelpers"
	pkgGRPC "github.com/stackrox/rox/pkg/grpc"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/allow"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"google.golang.org/grpc"
)

var (
	authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		allow.Anonymous(): {
			"/v1.ConfigService/GetPublicConfig",
		},
		user.With(permissions.View(resources.Config)): {
			"/v1.ConfigService/GetPrivateConfig",
		},
		user.With(permissions.View(resources.Config)): {
			"/v1.ConfigService/GetConfig",
		},
		user.With(permissions.Modify(resources.Config)): {
			"/v1.ConfigService/PutConfig",
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
func (s *serviceImpl) GetPublicConfig(ctx context.Context, _ *v1.Empty) (*storage.PublicConfig, error) {
	config, err := s.datastore.GetConfig(ctx)
	if err != nil {
		return nil, err
	}
	if config.GetPublicConfig() == nil {
		return &storage.PublicConfig{}, nil
	}
	return config.GetPublicConfig(), nil
}

// GetPrivateConfig returns the privately available config
func (s *serviceImpl) GetPrivateConfig(ctx context.Context, _ *v1.Empty) (*storage.PrivateConfig, error) {
	config, err := s.datastore.GetConfig(ctx)
	if err != nil {
		return nil, err
	}
	if config.GetPrivateConfig() == nil {
		return &storage.PrivateConfig{}, nil
	}
	return config.GetPrivateConfig(), nil
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
		return nil, errors.Wrap(errorhelpers.ErrInvalidArgs, "config must be specified")
	}
	if err := s.datastore.UpsertConfig(ctx, req.GetConfig()); err != nil {
		return nil, err
	}
	return req.GetConfig(), nil
}
