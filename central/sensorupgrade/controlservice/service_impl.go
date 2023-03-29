package service

import (
	"context"

	"github.com/gogo/protobuf/types"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/sensor/service/connection"
	"github.com/stackrox/rox/central/sensor/service/connection/upgradecontroller"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/grpc/authz/idcheck"
	"github.com/stackrox/rox/pkg/utils"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	authorizer = idcheck.SensorsOnly()
)

type service struct {
	central.UnimplementedSensorUpgradeControlServiceServer

	connectionManager connection.Manager
}

func clusterIDFromCtx(ctx context.Context) (string, error) {
	id, err := authn.IdentityFromContext(ctx)
	if err != nil {
		return "", err
	}

	svc := id.Service()
	if svc == nil || svc.GetType() != storage.ServiceType_SENSOR_SERVICE {
		return "", errox.NotAuthorized.CausedBy("only sensor/upgrader may access this API")
	}

	clusterID := svc.GetId()
	if clusterID == "" {
		return "", errox.NotAuthorized.CausedBy("only sensors with a valid cluster ID may access this API")
	}
	return clusterID, nil
}

func (s *service) UpgradeCheckInFromUpgrader(ctx context.Context, req *central.UpgradeCheckInFromUpgraderRequest) (*central.UpgradeCheckInFromUpgraderResponse, error) {
	clusterIDFromCert, err := clusterIDFromCtx(ctx)
	if err != nil {
		return nil, err
	}

	clusterID, err := centralsensor.GetClusterID(req.GetClusterId(), clusterIDFromCert)
	if err != nil {
		return nil, errors.Wrapf(errox.InvalidArgs, "failed to derive cluster ID: %s", err)
	}

	return s.connectionManager.ProcessCheckInFromUpgrader(ctx, clusterID, req)
}

func (s *service) UpgradeCheckInFromSensor(ctx context.Context, req *central.UpgradeCheckInFromSensorRequest) (*types.Empty, error) {
	clusterIDFromCert, err := clusterIDFromCtx(ctx)
	if err != nil {
		return nil, err
	}

	clusterID, err := centralsensor.GetClusterID(req.GetClusterId(), clusterIDFromCert)
	if err != nil {
		return nil, errors.Wrapf(errox.InvalidArgs, "failed to derive cluster ID: %s", err)
	}

	if err := s.connectionManager.ProcessUpgradeCheckInFromSensor(ctx, clusterID, req); err != nil {
		if errors.Is(err, upgradecontroller.ErrNoUpgradeInProgress) {
			s, err := status.New(codes.Internal, err.Error()).WithDetails(&central.UpgradeCheckInResponseDetails_NoUpgradeInProgress{})
			if utils.ShouldErr(err) == nil {
				return nil, s.Err()
			}
		}
		return nil, err
	}
	return &types.Empty{}, nil
}

func (s *service) RegisterServiceServer(server *grpc.Server) {
	central.RegisterSensorUpgradeControlServiceServer(server, s)
}

func (s *service) RegisterServiceHandler(_ context.Context, _ *runtime.ServeMux, _ *grpc.ClientConn) error {
	return nil
}

func (s *service) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, authorizer.Authorized(ctx, fullMethodName)
}
