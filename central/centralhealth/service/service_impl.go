package service

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/defaults/accesscontrol"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"github.com/stackrox/rox/pkg/migrations"
	"github.com/stackrox/rox/pkg/version"
	"google.golang.org/grpc"
)

var (
	authorizer = user.WithRole(accesscontrol.Admin)
)

type serviceImpl struct {
	v1.UnimplementedCentralHealthServiceServer
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *serviceImpl) RegisterServiceServer(grpcServer *grpc.Server) {
	v1.RegisterCentralHealthServiceServer(grpcServer, s)
}

// RegisterServiceHandler registers this service with the given gRPC Gateway endpoint.
func (s *serviceImpl) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return v1.RegisterCentralHealthServiceHandler(ctx, mux, conn)
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, authorizer.Authorized(ctx, fullMethodName)
}

// GetUpgradeStatus returns the upgrade status for Central.
func (s *serviceImpl) GetUpgradeStatus(_ context.Context, _ *v1.Empty) (*v1.GetUpgradeStatusResponse, error) {
	upgradeStatus := &v1.CentralUpgradeStatus{
		Version: version.GetMainVersion(),
		// Due to backwards compatibility going forward we can assume
		// we can rollback after an upgrade
		CanRollbackAfterUpgrade: true,
		ForceRollbackTo:         migrations.MinimumSupportedDBVersion(),
	}

	return &v1.GetUpgradeStatusResponse{
		UpgradeStatus: upgradeStatus,
	}, nil
}
