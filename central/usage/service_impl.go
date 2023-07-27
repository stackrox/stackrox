package usage

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/pkg/errors"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"github.com/stackrox/rox/pkg/sac/resources"
	"google.golang.org/grpc"
)

var (
	authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		user.With(permissions.View(resources.Administration)): {
			"/v1.UsageService/GetCurrentUsage",
			"/v1.UsageService/GetMaxUsage",
		}})
)

type serviceImpl struct {
	v1.UnimplementedUsageServiceServer
}

// New returns a new Service instance using the given DataStore.
func New(datastore any) Service {
	return &serviceImpl{}
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *serviceImpl) RegisterServiceServer(grpcServer *grpc.Server) {
	v1.RegisterUsageServiceServer(grpcServer, s)
}

// RegisterServiceHandler registers this service with the given gRPC Gateway endpoint.
func (s *serviceImpl) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return errors.Wrap(v1.RegisterUsageServiceHandler(ctx, mux, conn), "failed to register the usage service handler")
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, errors.Wrapf(authorizer.Authorized(ctx, fullMethodName), "failed to authorize a call to %s", fullMethodName)
}
