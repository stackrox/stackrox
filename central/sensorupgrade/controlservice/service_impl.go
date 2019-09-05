package service

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/stackrox/rox/central/sensor/service/connection"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/idcheck"
	"google.golang.org/grpc"
)

var (
	authorizer = idcheck.SensorsOnly()
)

type service struct {
	connectionManager connection.Manager
}

func clusterIDFromCtx(ctx context.Context) (string, error) {
	id := authn.IdentityFromContext(ctx)
	if id == nil {
		return "", authz.ErrNotAuthorized("no identity in context")
	}

	svc := id.Service()
	if svc == nil || svc.GetType() != storage.ServiceType_SENSOR_SERVICE {
		return "", authz.ErrNotAuthorized("only sensor/upgrader may access this API")
	}

	clusterID := svc.GetId()
	if clusterID == "" {
		return "", authz.ErrNotAuthorized("only sensors with a valid cluster ID may access this API")
	}
	return clusterID, nil
}

func (s *service) UpgradeCheckInFromUpgrader(ctx context.Context, req *central.UpgradeCheckInFromUpgraderRequest) (*central.UpgradeCheckInFromUpgraderResponse, error) {
	clusterID, err := clusterIDFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	return s.connectionManager.ProcessCheckInFromUpgrader(ctx, clusterID, req)
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
