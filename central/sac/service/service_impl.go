package service

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	errors "github.com/pkg/errors"
	"github.com/stackrox/default-authz-plugin/pkg/payload"
	"github.com/stackrox/rox/central/auth/userpass"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/central/sac/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/client"
	"github.com/stackrox/rox/pkg/secrets"
	"google.golang.org/grpc"
)

var (
	authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		user.With(permissions.View(resources.AuthPlugin)): {
			"/v1.ScopedAccessControlService/DryRunAuthzPluginConfig",
			"/v1.ScopedAccessControlService/GetAuthzPluginConfigs",
		},
		user.With(permissions.Modify(resources.AuthPlugin)): {
			"/v1.ScopedAccessControlService/AddAuthzPluginConfig",
			"/v1.ScopedAccessControlService/UpdateAuthzPluginConfig",
			"/v1.ScopedAccessControlService/DeleteAuthzPluginConfig",
		},
	})

	testPrincipal = payload.Principal{
		AuthProvider: payload.AuthProviderInfo{
			ID:   "test_id",
			Type: "test_type",
			Name: "test_name",
		},
		Attributes: map[string]interface{}{
			"user": []string{"test_user"},
		},
	}

	testScope = payload.AccessScope{
		Verb: sac.AccessModeScopeKey(storage.Access_READ_ACCESS).Verb(),
		Noun: sac.ResourceScopeKey(resources.Cluster.GetResource()).String(),
		Attributes: payload.NounAttributes{
			Cluster: payload.Cluster{
				ID:   "test_cluster_id",
				Name: "test_cluster_name",
			},
			Namespace: "test_namespace",
		},
	}
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
	if err := validateConfig(req.GetConfig()); err != nil {
		return nil, errors.Wrap(errox.InvalidArgs, err.Error())
	}
	if _, err := s.reconcileUpsertAuthzPluginConfigRequest(ctx, req); err != nil {
		return nil, errors.Wrap(errox.InvalidArgs, err.Error())
	}
	if err := s.testConfig(ctx, req.GetConfig()); err != nil {
		return nil, errors.Wrap(errox.InvalidArgs, err.Error())
	}
	return &v1.Empty{}, nil
}

func (s *serviceImpl) GetAuthzPluginConfigs(ctx context.Context, _ *v1.Empty) (*v1.GetAuthzPluginConfigsResponse, error) {
	configs, err := s.ds.ListAuthzPluginConfigs(ctx)
	if err != nil {
		return nil, err
	}
	for _, config := range configs {
		secrets.ScrubSecretsFromStructWithReplacement(config, secrets.ScrubReplacementStr)
	}
	return &v1.GetAuthzPluginConfigsResponse{
		Configs: configs,
	}, nil
}

func (s *serviceImpl) AddAuthzPluginConfig(ctx context.Context, req *v1.UpsertAuthzPluginConfigRequest) (*storage.AuthzPluginConfig, error) {
	cfg := req.GetConfig()

	if err := validateConfig(cfg); err != nil {
		return nil, errors.Wrap(errox.InvalidArgs, err.Error())
	}
	if cfg.GetEnabled() {
		if err := s.testConfig(ctx, cfg); err != nil {
			return nil, errors.Wrapf(errox.InvalidArgs, "%v\nCheck the central logs for full error.", err)
		}
	}

	// Allow modifying enabled plugin only for basic auth user.
	if userpass.IsLocalAdmin(authn.IdentityFromContextOrNil(ctx)) {
		ctx = datastore.WithModifyEnabledPluginCap(ctx)
	}

	cfg.Id = "" // add
	upsertedConfig, err := s.ds.UpsertAuthzPluginConfig(ctx, cfg)
	if err != nil {
		return nil, err
	}
	return upsertedConfig, nil
}

func (s *serviceImpl) UpdateAuthzPluginConfig(ctx context.Context, req *v1.UpsertAuthzPluginConfigRequest) (*storage.AuthzPluginConfig, error) {
	cfg := req.GetConfig()

	if cfg.GetId() == "" {
		return nil, errors.Wrap(errox.InvalidArgs, "update must specify an ID")
	}

	if err := validateConfig(cfg); err != nil {
		return nil, errors.Wrap(errox.InvalidArgs, err.Error())
	}
	reconciled, err := s.reconcileUpsertAuthzPluginConfigRequest(ctx, req)
	if err != nil {
		return nil, errors.Wrap(errox.InvalidArgs, err.Error())
	}
	if cfg.GetEnabled() {
		if err := s.testConfig(ctx, cfg); err != nil {
			return nil, errors.Wrapf(errox.InvalidArgs, "%v\nCheck the central logs for full error.", err)
		}
	}

	// Allow modifying enabled plugin only for basic auth user.
	if userpass.IsLocalAdmin(authn.IdentityFromContextOrNil(ctx)) {
		ctx = datastore.WithModifyEnabledPluginCap(ctx)
	}

	upsertedConfig, err := s.ds.UpsertAuthzPluginConfig(ctx, cfg)
	if err != nil {
		return nil, err
	}
	if reconciled {
		secrets.ScrubSecretsFromStructWithReplacement(upsertedConfig, secrets.ScrubReplacementStr)
	}
	return upsertedConfig, nil
}

func (s *serviceImpl) DeleteAuthzPluginConfig(ctx context.Context, req *v1.ResourceByID) (*v1.Empty, error) {
	// Allow modifying enabled plugin only for basic auth user.
	if userpass.IsLocalAdmin(authn.IdentityFromContextOrNil(ctx)) {
		ctx = datastore.WithModifyEnabledPluginCap(ctx)
	}

	if err := s.ds.DeleteAuthzPluginConfig(ctx, req.GetId()); err != nil {
		return nil, err
	}
	return &v1.Empty{}, nil
}

func (s *serviceImpl) testConfig(ctx context.Context, config *storage.AuthzPluginConfig) error {
	newClient, err := client.New(config.GetEndpointConfig())
	if err != nil {
		return err
	}

	_, _, err = newClient.ForUser(ctx, testPrincipal, testScope)
	if err != nil {
		return err
	}
	return nil
}

func (s *serviceImpl) reconcileUpsertAuthzPluginConfigRequest(ctx context.Context, updateRequest *v1.UpsertAuthzPluginConfigRequest) (bool, error) {
	if updateRequest.GetUpdatePassword() {
		return false, nil
	}
	if updateRequest.GetConfig() == nil {
		return false, errors.Wrap(errox.InvalidArgs, "request is missing authz plugin config")
	}
	if updateRequest.GetConfig().GetId() == "" {
		return false, errors.Wrap(errox.NotFound, "id required for stored credential reconciliation")
	}
	existingAuthzPluginConfig, err := s.ds.GetAuthzPluginConfig(ctx, updateRequest.GetConfig().GetId())
	if err != nil {
		return false, err
	}
	if existingAuthzPluginConfig == nil {
		return false, errors.Wrapf(errox.NotFound, "existing authz plugin %s not found", updateRequest.GetConfig().GetId())
	}
	if err := reconcileAuthzPluginConfigWithExisting(updateRequest.GetConfig(), existingAuthzPluginConfig); err != nil {
		return false, errors.Wrap(errox.InvalidArgs, err.Error())
	}
	return true, nil
}

func reconcileAuthzPluginConfigWithExisting(updated, existing *storage.AuthzPluginConfig) error {
	return secrets.ReconcileScrubbedStructWithExisting(updated, existing)
}
