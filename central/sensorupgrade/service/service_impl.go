package service

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/central/sensor/service/connection"
	"github.com/stackrox/rox/central/sensorupgradeconfig/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"google.golang.org/grpc"
)

var (
	authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		user.With(permissions.View(resources.SensorUpgradeConfig)): {
			"/v1.SensorUpgradeService/GetSensorUpgradeConfig",
		},
		user.With(permissions.Modify(resources.SensorUpgradeConfig)): {
			"/v1.SensorUpgradeService/UpdateSensorUpgradeConfig",
		},
		user.With(permissions.Modify(resources.Cluster)): {
			"/v1.SensorUpgradeService/TriggerSensorUpgrade",
			"/v1.SensorUpgradeService/TriggerSensorCertRotation",
		},
	})
)

type service struct {
	configDataStore datastore.DataStore
	manager         connection.Manager
}

func (s *service) RegisterServiceServer(server *grpc.Server) {
	v1.RegisterSensorUpgradeServiceServer(server, s)
}

func (s *service) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return v1.RegisterSensorUpgradeServiceHandler(ctx, mux, conn)
}

func (s *service) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, authorizer.Authorized(ctx, fullMethodName)
}

func (s *service) GetSensorUpgradeConfig(ctx context.Context, _ *v1.Empty) (*v1.GetSensorUpgradeConfigResponse, error) {
	config, err := s.configDataStore.GetSensorUpgradeConfig(ctx)
	if err != nil {
		return nil, err
	}
	if config == nil {
		return nil, errors.Wrap(errox.NotFound, "couldn't find sensor upgrade config")
	}
	return &v1.GetSensorUpgradeConfigResponse{Config: config}, nil
}

func (s *service) UpdateSensorUpgradeConfig(ctx context.Context, req *v1.UpdateSensorUpgradeConfigRequest) (*v1.Empty, error) {
	if req.GetConfig() == nil {
		return nil, errors.Wrap(errox.InvalidArgs, "need to specify a config")
	}
	if err := s.configDataStore.UpsertSensorUpgradeConfig(ctx, req.GetConfig()); err != nil {
		return nil, err
	}
	return &v1.Empty{}, nil
}

func (s *service) TriggerSensorUpgrade(ctx context.Context, req *v1.ResourceByID) (*v1.Empty, error) {
	if req.GetId() == "" {
		return nil, errors.Wrap(errox.InvalidArgs, "no cluster ID specified")
	}

	err := s.manager.TriggerUpgrade(ctx, req.GetId())
	if err != nil {
		return nil, err
	}
	return &v1.Empty{}, nil
}
func (s *service) TriggerSensorCertRotation(ctx context.Context, req *v1.ResourceByID) (*v1.Empty, error) {
	if req.GetId() == "" {
		return nil, errors.Wrap(errox.InvalidArgs, "no cluster ID specified")
	}

	err := s.manager.TriggerCertRotation(ctx, req.GetId())
	if err != nil {
		return nil, err
	}
	return &v1.Empty{}, nil

}
