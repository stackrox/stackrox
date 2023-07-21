package service

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/sensor/service/connection"
	"github.com/stackrox/rox/central/sensorupgradeconfig/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"google.golang.org/grpc"
)

var (
	authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		user.With(permissions.View(resources.Administration)): {
			"/v1.SensorUpgradeService/GetSensorUpgradeConfig",
		},
		user.With(permissions.Modify(resources.Administration)): {
			"/v1.SensorUpgradeService/UpdateSensorUpgradeConfig",
		},
		user.With(permissions.Modify(resources.Cluster)): {
			"/v1.SensorUpgradeService/TriggerSensorUpgrade",
			"/v1.SensorUpgradeService/TriggerSensorCertRotation",
		},
	})
)

type service struct {
	v1.UnimplementedSensorUpgradeServiceServer

	configDataStore datastore.DataStore
	manager         connection.Manager
	autoTriggerFlag concurrency.Flag
}

func (s *service) initialize() error {
	ctx := sac.WithAllAccess(context.Background())
	defaultConfig, err := s.getOrCreateSensorUpgradeConfig(ctx)
	if err != nil {
		return err
	}
	s.autoTriggerFlag.Set(defaultConfig.EnableAutoUpgrade)
	return nil
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

func (s *service) wrapToggleResponse(config *storage.SensorUpgradeConfig) *v1.GetSensorUpgradeConfigResponse {
	return &v1.GetSensorUpgradeConfigResponse{
		Config: &v1.GetSensorUpgradeConfigResponse_UpgradeConfig{
			EnableAutoUpgrade:  config.GetEnableAutoUpgrade(),
			AutoUpgradeFeature: getAutoUpgradeFeatureStatus(),
		},
	}
}

// getOrCreateSensorUpgradeConfig returns the upgrade config stored in the DB. If there's no entry
// in the DB, create one based on the default value.
func (s *service) getOrCreateSensorUpgradeConfig(ctx context.Context) (*storage.SensorUpgradeConfig, error) {
	config, err := s.configDataStore.GetSensorUpgradeConfig(ctx)
	if err != nil {
		return nil, err
	}
	if config == nil {
		// If there's no config in the DB, return default config according to managed central flag
		// and insert the value
		if getAutoUpgradeFeatureStatus() == v1.GetSensorUpgradeConfigResponse_SUPPORTED {
			config = &storage.SensorUpgradeConfig{EnableAutoUpgrade: true}
		} else {
			config = &storage.SensorUpgradeConfig{EnableAutoUpgrade: false}
		}
		if err := s.configDataStore.UpsertSensorUpgradeConfig(ctx, config); err != nil {
			return nil, err
		}
	}
	return config, nil
}

func (s *service) GetSensorUpgradeConfig(ctx context.Context, _ *v1.Empty) (*v1.GetSensorUpgradeConfigResponse, error) {
	config, err := s.configDataStore.GetSensorUpgradeConfig(ctx)
	if err != nil {
		return nil, err
	}
	if config == nil {
		return nil, errors.Wrap(errox.NotFound, "couldn't find sensor upgrade config")
	}
	return s.wrapToggleResponse(config), nil
}

func (s *service) AutoUpgradeSetting() *concurrency.Flag {
	return &s.autoTriggerFlag
}

func getAutoUpgradeFeatureStatus() v1.GetSensorUpgradeConfigResponse_SensorAutoUpgradeFeatureStatus {
	if env.ManagedCentral.BooleanSetting() {
		return v1.GetSensorUpgradeConfigResponse_NOT_SUPPORTED
	}
	return v1.GetSensorUpgradeConfigResponse_SUPPORTED
}

func (s *service) UpdateSensorUpgradeConfig(ctx context.Context, req *v1.UpdateSensorUpgradeConfigRequest) (*v1.Empty, error) {
	if req.GetConfig() == nil {
		return nil, errors.Wrap(errox.InvalidArgs, "need to specify a config")
	}

	if req.GetConfig().GetEnableAutoUpgrade() && getAutoUpgradeFeatureStatus() == v1.GetSensorUpgradeConfigResponse_NOT_SUPPORTED {
		return nil, errors.Wrap(errox.InvalidArgs, "auto-upgrade not supported on managed ACS")
	}

	if err := s.configDataStore.UpsertSensorUpgradeConfig(ctx, req.GetConfig()); err != nil {
		return nil, err
	}
	s.autoTriggerFlag.Set(req.GetConfig().EnableAutoUpgrade)
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
