package service

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/allow"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"google.golang.org/grpc"
)

var (
	authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		allow.Anonymous(): {
			"/v1.TargetClusterVersionService/GetTargetClusterVersion",
		},
	})
)

type serviceImpl struct {
	v1.UnimplementedTargetClusterVersionServiceServer
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *serviceImpl) RegisterServiceServer(grpcServer *grpc.Server) {
	v1.RegisterTargetClusterVersionServiceServer(grpcServer, s)
}

// RegisterServiceHandler registers this service with the given gRPC Gateway endpoint.
func (s *serviceImpl) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return v1.RegisterTargetClusterVersionServiceHandler(ctx, mux, conn)
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, authorizer.Authorized(ctx, fullMethodName)
}

func (s *serviceImpl) GetTargetClusterVersion(ctx context.Context, _ *v1.Empty) (*v1.TargetClusterVersionResponse, error) {
	targetVersion := &v1.TargetClusterVersionResponse{
		Version: "42.42.24",
		// Version: version.GetMainVersion // as a default. Pull env var for PoC for now.
	}
	return targetVersion, nil
}
