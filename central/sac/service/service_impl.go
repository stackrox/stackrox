package service

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/central/sac/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		user.With(permissions.View(resources.AuthPlugin)): {
			"/v1.ScopedAccessControlService/DryRunAuthzPluginConfig",
			"/v1.ScopedAccessControlService/GetAuthzPluginConfigs",
		},
		user.With(permissions.Modify(resources.AuthPlugin)): {
			"/v1.ScopedAccessControlService/ConfigureAuthzPlugin",
			"/v1.ScopedAccessControlService/DeleteAuthzPlugin",
		},
	})
)

type serviceImpl struct {
	ds datastore.DataStore
}

func (s *serviceImpl) RegisterServiceServer(grpcServer *grpc.Server) {
	v1.RegisterScopedAccessControlServiceServer(grpcServer, s)
}

func (s *serviceImpl) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return v1.RegisterScopedAccessControlServiceHandler(ctx, mux, conn)
}

func (*serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, authorizer.Authorized(ctx, fullMethodName)
}

func (s *serviceImpl) DryRunAuthzPluginConfig(ctx context.Context, req *v1.UpsertAuthzPluginConfigRequest) (*v1.Empty, error) {
	// Build client.

	// Test

	return nil, status.Error(codes.Unimplemented, "Unimplemented")
}

func (s *serviceImpl) GetAuthzPluginConfigs(ctx context.Context, _ *v1.Empty) (*v1.GetAuthzPluginConfigsResponse, error) {
	configs, err := s.ds.ListAuthzPluginConfigs(ctx)
	if err != nil {
		return nil, err
	}
	return &v1.GetAuthzPluginConfigsResponse{
		Configs: configs,
	}, nil
}

func (s *serviceImpl) ConfigureAuthzPlugin(ctx context.Context, req *v1.UpsertAuthzPluginConfigRequest) (*storage.AuthzPluginConfig, error) {
	config, err := s.ds.UpsertAuthzPluginConfig(ctx, req.GetConfig())
	if err != nil {
		return nil, err
	}
	return config, nil
}

func (s *serviceImpl) DeleteAuthzPlugin(ctx context.Context, req *v1.ResourceByID) (*v1.Empty, error) {
	if err := s.ds.DeleteAuthzPluginConfig(ctx, req.GetId()); err != nil {
		return nil, err
	}
	return &v1.Empty{}, nil
}
