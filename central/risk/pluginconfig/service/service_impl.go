package service

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/stackrox/rox/central/risk/scorer/plugin"
	"github.com/stackrox/rox/central/risk/scorer/plugin/registry"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sac/resources"
	"google.golang.org/grpc"
)

var (
	log = logging.LoggerForModule()

	authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		user.With(permissions.View(resources.DeploymentExtension)): {
			v1.RiskScoringPluginService_ListRiskScoringPluginConfigs_FullMethodName,
			v1.RiskScoringPluginService_GetRiskScoringPluginConfig_FullMethodName,
		},
		user.With(permissions.Modify(resources.DeploymentExtension)): {
			v1.RiskScoringPluginService_UpsertRiskScoringPluginConfig_FullMethodName,
			v1.RiskScoringPluginService_DeleteRiskScoringPluginConfig_FullMethodName,
		},
	})
)

type serviceImpl struct {
	v1.UnimplementedRiskScoringPluginServiceServer

	registry registry.Registry
}

func (s *serviceImpl) RegisterServiceServer(grpcServer *grpc.Server) {
	v1.RegisterRiskScoringPluginServiceServer(grpcServer, s)
}

func (s *serviceImpl) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return v1.RegisterRiskScoringPluginServiceHandler(ctx, mux, conn)
}

func (*serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, authorizer.Authorized(ctx, fullMethodName)
}

func (s *serviceImpl) ListRiskScoringPluginConfigs(ctx context.Context, _ *v1.Empty) (*v1.ListRiskScoringPluginConfigsResponse, error) {
	plugins := s.registry.GetEnabledPlugins()
	configs := make([]*storage.RiskScoringPluginConfig, 0, len(plugins))
	for _, p := range plugins {
		configs = append(configs, configToProto(p.Config))
	}

	// Also include disabled configs
	allConfigs := s.registry.GetAllConfigs()
	for _, cfg := range allConfigs {
		if !cfg.Enabled {
			configs = append(configs, configToProto(cfg))
		}
	}

	return &v1.ListRiskScoringPluginConfigsResponse{Configs: configs}, nil
}

func (s *serviceImpl) GetRiskScoringPluginConfig(ctx context.Context, req *v1.GetRiskScoringPluginConfigRequest) (*storage.RiskScoringPluginConfig, error) {
	cfg, ok := s.registry.GetConfig(req.GetId())
	if !ok {
		return nil, errox.NotFound.Newf("plugin config %q not found", req.GetId())
	}
	return configToProto(cfg), nil
}

func (s *serviceImpl) UpsertRiskScoringPluginConfig(ctx context.Context, req *v1.UpsertRiskScoringPluginConfigRequest) (*v1.UpsertRiskScoringPluginConfigResponse, error) {
	cfg := protoToConfig(req.GetConfig())
	if err := s.registry.UpsertConfig(cfg); err != nil {
		return nil, err
	}
	log.Infof("Upserted plugin config: %s", cfg.ID)
	return &v1.UpsertRiskScoringPluginConfigResponse{Config: req.GetConfig()}, nil
}

func (s *serviceImpl) DeleteRiskScoringPluginConfig(ctx context.Context, req *v1.DeleteRiskScoringPluginConfigRequest) (*v1.Empty, error) {
	if err := s.registry.DeleteConfig(req.GetId()); err != nil {
		return nil, err
	}
	log.Infof("Deleted plugin config: %s", req.GetId())
	return &v1.Empty{}, nil
}

// configToProto converts a plugin.Config to storage.RiskScoringPluginConfig.
func configToProto(cfg *plugin.Config) *storage.RiskScoringPluginConfig {
	return &storage.RiskScoringPluginConfig{
		Id:       cfg.ID,
		Name:     cfg.Name,
		Type:     storage.PluginType(cfg.Type),
		Enabled:  cfg.Enabled,
		Weight:   cfg.Weight,
		Priority: cfg.Priority,
		Builtin: &storage.BuiltinPluginConfig{
			PluginName: cfg.Name,
			Parameters: cfg.Parameters,
		},
	}
}

// protoToConfig converts a storage.RiskScoringPluginConfig to plugin.Config.
func protoToConfig(proto *storage.RiskScoringPluginConfig) *plugin.Config {
	return &plugin.Config{
		ID:         proto.GetId(),
		Name:       proto.GetBuiltin().GetPluginName(),
		Type:       plugin.PluginType(proto.GetType()),
		Enabled:    proto.GetEnabled(),
		Weight:     proto.GetWeight(),
		Priority:   proto.GetPriority(),
		Parameters: proto.GetBuiltin().GetParameters(),
	}
}
