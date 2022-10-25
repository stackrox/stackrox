package service

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/grpc/authz/allow"
	"google.golang.org/grpc"
)

type serviceImpl struct {
	v1.UnimplementedPingServiceServer
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *serviceImpl) RegisterServiceServer(grpcServer *grpc.Server) {
	v1.RegisterPingServiceServer(grpcServer, s)
}

// RegisterServiceHandler registers this service with the given gRPC Gateway endpoint.
func (s *serviceImpl) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return v1.RegisterPingServiceHandler(ctx, mux, conn)
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, allow.Anonymous().Authorized(ctx, fullMethodName)
}

// Ping implements v1.PingServiceServer, and it always returns a v1.PongMessage object.
func (s *serviceImpl) Ping(context.Context, *v1.Empty) (*v1.PongMessage, error) {
	result := &v1.PongMessage{
		Status: "ok",
	}
	return result, nil
}
