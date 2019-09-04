package service

import (
	"context"

	"github.com/gogo/protobuf/types"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/stackrox/rox/central/sensor/service/connection"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/idcheck"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	authorizer = idcheck.SensorsOnly()
)

type service struct {
	connectionManager connection.Manager
}

func (s *service) RecordUpgradeProgress(ctx context.Context, req *central.RecordUpgradeProgressRequest) (*types.Empty, error) {
	id := authn.IdentityFromContext(ctx)
	if id == nil {
		return nil, authz.ErrNotAuthorized("no identity in context")
	}

	svc := id.Service()
	if svc == nil || svc.GetType() != storage.ServiceType_SENSOR_SERVICE {
		return nil, authz.ErrNotAuthorized("only sensor/upgrader may access this API")
	}

	clusterID := svc.GetId()
	if clusterID == "" {
		return nil, authz.ErrNotAuthorized("only sensors with a valid cluster ID may access this API")
	}

	if err := s.connectionManager.RecordUpgradeProgress(clusterID, req.GetUpgradeProcessId(), req.GetUpgradeProgress()); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.Empty{}, nil
}

func (s *service) RegisterServiceServer(server *grpc.Server) {
	central.RegisterSensorUpgradeControlServiceServer(server, s)
}

func (s *service) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return nil
}

func (s *service) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, authorizer.Authorized(ctx, fullMethodName)
}
