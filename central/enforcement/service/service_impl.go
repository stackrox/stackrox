package service

import (
	"context"
	"fmt"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/central/sensorevent/service/streamer"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"google.golang.org/grpc"
)

var (
	authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		user.With(permissions.Modify(resources.Enforcements)): {
			"/v1.EnforcementService/ApplyEnforcement",
		},
	})
)

// serviceImpl provides APIs for alerts.
type serviceImpl struct {
	manager streamer.Manager
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *serviceImpl) RegisterServiceServer(grpcServer *grpc.Server) {
	v1.RegisterEnforcementServiceServer(grpcServer, s)
}

// RegisterServiceHandler registers this service with the given gRPC Gateway endpoint.
func (s *serviceImpl) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return v1.RegisterEnforcementServiceHandler(ctx, mux, conn)
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, authorizer.Authorized(ctx, fullMethodName)
}

// GetSecret returns the secret for the id.
func (s *serviceImpl) ApplyEnforcement(ctx context.Context, request *v1.EnforcementRequest) (*v1.Empty, error) {
	activeStream := s.manager.GetStreamer(request.GetClusterId())
	if activeStream == nil {
		return &v1.Empty{}, fmt.Errorf("cluster %s not available", request.GetClusterId())
	}

	if !activeStream.InjectEnforcement(request.GetEnforcement()) {
		return &v1.Empty{}, fmt.Errorf("unable to push enforcement to cluster %s", request.GetClusterId())
	}
	return &v1.Empty{}, nil
}
