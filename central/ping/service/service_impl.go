package service

import (
	"context"
	"sync/atomic"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/pkg/errors"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/allow"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"google.golang.org/grpc"
)

var (
	authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		allow.Anonymous(): {
			// The Ping endpoint is not process intensive, and does not expose
			// any sensitive information (it only returns a hardcoded value).
			// Changing from public to authenticated would actually make the
			// associated process heavier. Therefore the endpoint can remain
			// public.
			v1.PingService_Ping_FullMethodName,
		},
	})

	// IsLeader is set by main package to indicate leader status
	IsLeader atomic.Bool
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
	return ctx, authorizer.Authorized(ctx, fullMethodName)
}

// Ping implements v1.PingServiceServer, and returns a v1.PongMessage object if this instance is the leader.
func (s *serviceImpl) Ping(context.Context, *v1.Empty) (*v1.PongMessage, error) {
	if !IsLeader.Load() {
		return nil, errors.New("not leader")
	}
	result := &v1.PongMessage{
		Status: "ok",
	}
	return result, nil
}
