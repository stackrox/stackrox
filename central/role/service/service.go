package service

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/stackrox/rox/central/role/store"
	"github.com/stackrox/rox/generated/api/v1"
	"google.golang.org/grpc"
)

// Service provides the interface to the gRPC service for roles.
type Service interface {
	RegisterServiceServer(grpcServer *grpc.Server)
	RegisterServiceHandlerFromEndpoint(ctx context.Context, mux *runtime.ServeMux, endpoint string, opts []grpc.DialOption) error

	AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error)
	v1.RoleServiceServer
}

// New returns a new instance of the service. Please use the Singleton instead.
func New(roleStore store.Store) Service {
	return &serviceImpl{
		roleStore: roleStore,
	}
}
