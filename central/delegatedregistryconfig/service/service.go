package service

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/delegatedregistryconfig/datastore"
	"github.com/stackrox/rox/central/role/resources"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/errox"
	pkgGRPC "github.com/stackrox/rox/pkg/grpc"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/or"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"github.com/stackrox/rox/pkg/logging"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	log        = logging.LoggerForModule()
	authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		or.SensorOrAuthorizer(user.With(permissions.View(resources.Administration))): {
			"/v1.DelegatedRegistryConfigService/GetConfig",
		},
		user.With(permissions.Modify(resources.Administration)): {
			"/v1.DelegatedRegistryConfigService/PutConfig",
		},
	})
)

// Service provides the interface to modify the delegated registry config
type Service interface {
	pkgGRPC.APIService

	AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error)

	v1.DelegatedRegistryConfigServiceServer
}

// New returns a new Service instance using the given DataStore.
func New(datastore datastore.DataStore) Service {
	return &serviceImpl{
		datastore: datastore,
	}
}

type serviceImpl struct {
	v1.UnimplementedDelegatedRegistryConfigServiceServer

	datastore datastore.DataStore
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *serviceImpl) RegisterServiceServer(grpcServer *grpc.Server) {
	v1.RegisterDelegatedRegistryConfigServiceServer(grpcServer, s)
}

// RegisterServiceHandler registers this service with the given gRPC Gateway endpoint.
func (s *serviceImpl) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return v1.RegisterDelegatedRegistryConfigServiceHandler(ctx, mux, conn)
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, authorizer.Authorized(ctx, fullMethodName)
}

// GetConfig returns Central's delegated registry config
func (s *serviceImpl) GetConfig(ctx context.Context, _ *v1.Empty) (*storage.DelegatedRegistryConfig, error) {
	if s.datastore == nil {
		return nil, status.Errorf(codes.Unimplemented, "datastore not initialized, is postgres enabled?")
	}

	config, err := s.datastore.GetConfig(ctx)
	if err != nil {
		return nil, err
	}

	if config == nil {
		return &storage.DelegatedRegistryConfig{}, nil
	}

	return config, nil
}

// PutConfig updates Central's delegated registry config
func (s *serviceImpl) PutConfig(ctx context.Context, config *storage.DelegatedRegistryConfig) (*storage.DelegatedRegistryConfig, error) {
	if s.datastore == nil {
		return nil, status.Errorf(codes.Unimplemented, "datastore not initialized, is postgres enabled?")
	}

	if config == nil {
		return nil, errors.Wrap(errox.InvalidArgs, "config must be specified")
	}

	if err := s.datastore.UpsertConfig(ctx, config); err != nil {
		return nil, err
	}

	return config, nil
}
