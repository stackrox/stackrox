package service

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/stackrox/rox/central/continuousintegration/datastore"
	"github.com/stackrox/rox/central/continuousintegration/token"
	"github.com/stackrox/rox/central/role/resources"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/allow"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"google.golang.org/grpc"
)

var (
	authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		user.With(permissions.View(resources.Integration)): {
			"/v1.ContinuousIntegrationService/GetContinuousIntegration",
			"/v1.ContinuousIntegrationService/ListContinuousIntegrations",
		},
		user.With(permissions.Modify(resources.Integration)): {
			"/v1.ContinuousIntegrationService/PostContinuousIntegration",
			"/v1.ContinuousIntegrationService/DeleteContinuousIntegration",
		},
		allow.Anonymous(): {
			"/v1.ContinuousIntegrationService/RetrieveTokenForContinuousIntegration",
		},
	})

	_ v1.ContinuousIntegrationServiceServer = (*serviceImpl)(nil)
)

type serviceImpl struct {
	v1.UnimplementedContinuousIntegrationServiceServer

	dataStore datastore.DataStore
	exchanger token.Exchanger
}

func (s *serviceImpl) RegisterServiceServer(grpcServer *grpc.Server) {
	v1.RegisterContinuousIntegrationServiceServer(grpcServer, s)
}

// RegisterServiceHandler registers this service with the given gRPC Gateway endpoint.
func (s *serviceImpl) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return v1.RegisterContinuousIntegrationServiceHandler(ctx, mux, conn)
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, authorizer.Authorized(ctx, fullMethodName)
}

func (s *serviceImpl) GetContinuousIntegration(ctx context.Context, req *v1.ResourceByID) (*storage.ContinuousIntegrationConfig, error) {
	cfg, _, err := s.dataStore.GetContinuousIntegrationConfig(ctx, req.GetId())
	return cfg, err
}

func (s *serviceImpl) ListContinuousIntegrations(ctx context.Context, _ *v1.Empty) (*v1.GetContinuousIntegrationConfigsResponse, error) {
	cfgs, err := s.dataStore.GetAllContinuousIntegrationConfigs(ctx)
	if err != nil {
		return nil, err
	}
	return &v1.GetContinuousIntegrationConfigsResponse{Configs: cfgs}, nil
}

func (s *serviceImpl) PostContinuousIntegration(ctx context.Context, cfg *storage.ContinuousIntegrationConfig) (*storage.ContinuousIntegrationConfig, error) {
	createdCfg, err := s.dataStore.AddContinuousIntegrationConfig(ctx, cfg)
	return createdCfg, err
}

func (s *serviceImpl) DeleteContinuousIntegration(ctx context.Context, req *v1.ResourceByID) (*v1.Empty, error) {
	if err := s.dataStore.RemoveContinuousIntegrationConfig(ctx, req.GetId()); err != nil {
		return nil, err
	}
	return &v1.Empty{}, nil
}

func (s *serviceImpl) RetrieveTokenForContinuousIntegration(ctx context.Context,
	req *v1.RetrieveTokenForContinuousIntegrationRequest) (*v1.RetrieveTokenForContinuousIntegrationResponse, error) {
	accessToken, err := s.exchanger.ExchangeToken(ctx, req.GetIdToken(), req.GetCiProvider())
	if err != nil {
		return nil, err
	}
	return &v1.RetrieveTokenForContinuousIntegrationResponse{
		AccessToken: accessToken,
	}, nil
}
