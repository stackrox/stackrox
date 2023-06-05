package service

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/stackrox/rox/central/declarativeconfig/health/datastore"
	"github.com/stackrox/rox/central/role/resources"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"google.golang.org/grpc"
)

var (
	authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		user.With(permissions.View(resources.Integration)): {
			"/v1.DeclarativeConfigHealthService/GetDeclarativeConfigHealths",
			"/v1.DeclarativeConfigHealthService/GetDeclarativeConfigHealth",
		},
	})

	_ v1.DeclarativeConfigHealthServiceServer = (*serviceImpl)(nil)
)

type serviceImpl struct {
	v1.UnimplementedDeclarativeConfigHealthServiceServer

	datastore datastore.DataStore
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *serviceImpl) RegisterServiceServer(server *grpc.Server) {
	v1.RegisterDeclarativeConfigHealthServiceServer(server, s)
}

// RegisterServiceHandler registers this service with the given gRPC Gateway endpoint.
func (s *serviceImpl) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return v1.RegisterDeclarativeConfigHealthServiceHandler(ctx, mux, conn)
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, authorizer.Authorized(ctx, fullMethodName)
}

// GetDeclarativeConfigHealths returns all declarative config healths currently available.
func (s *serviceImpl) GetDeclarativeConfigHealths(ctx context.Context, _ *v1.Empty) (*v1.GetDeclarativeConfigHealthsResponse, error) {
	healths, err := s.datastore.GetDeclarativeConfigs(ctx)
	if err != nil {
		return nil, err
	}
	return &v1.GetDeclarativeConfigHealthsResponse{Healths: healths}, nil
}

// GetDeclarativeConfigHealth returns a specific declarative config health by ID.
func (s *serviceImpl) GetDeclarativeConfigHealth(ctx context.Context, req *v1.ResourceByID) (*v1.GetDeclarativeConfigHealthResponse, error) {
	health, exists, err := s.datastore.GetDeclarativeConfig(ctx, req.GetId())
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errox.NotFound.Newf("declarative config health %q does not exists", req.GetId())
	}
	return &v1.GetDeclarativeConfigHealthResponse{Health: health}, nil
}
