package service

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/stackrox/rox/central/apitoken"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/central/role/store"
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
		user.With(permissions.View(resources.APIToken)): {
			"/v1.APITokenService/GetAPIToken",
			"/v1.APITokenService/GetAPITokens",
		},
		user.With(permissions.Modify(resources.APIToken)): {
			"/v1.APITokenService/GenerateToken",
			"/v1.APITokenService/RevokeToken",
		},
	})
)

type serviceImpl struct {
	backend   apitoken.Backend
	roleStore store.Store
}

func (s *serviceImpl) GetAPIToken(ctx context.Context, req *v1.ResourceByID) (*storage.TokenMetadata, error) {
	if req.GetId() == "" {
		return nil, status.Error(codes.InvalidArgument, "empty id passed")
	}
	token, err := s.backend.GetTokenOrNil(req.GetId())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "token retrieval failed: %s", err)
	}
	if token == nil {
		return nil, status.Errorf(codes.InvalidArgument, "token with id '%s' does not exist", req.GetId())
	}
	return token, nil
}

func (s *serviceImpl) GetAPITokens(ctx context.Context, req *v1.GetAPITokensRequest) (*v1.GetAPITokensResponse, error) {
	tokens, err := s.backend.GetTokens(req)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "retrieval of tokens failed: %s", err)
	}
	return &v1.GetAPITokensResponse{
		Tokens: tokens,
	}, nil
}

func (s *serviceImpl) RevokeToken(ctx context.Context, req *v1.ResourceByID) (*v1.Empty, error) {
	exists, err := s.backend.RevokeToken(req.GetId())
	if err != nil {
		return &v1.Empty{}, status.Errorf(codes.Internal, "couldn't revoke token: %s", err)
	}
	if !exists {
		return &v1.Empty{}, status.Errorf(codes.Internal, "token with id '%s' does not exist", req.GetId())
	}
	return &v1.Empty{}, nil
}

func (s *serviceImpl) GenerateToken(ctx context.Context, req *v1.GenerateTokenRequest) (*v1.GenerateTokenResponse, error) {
	if req.GetName() == "" {
		return nil, status.Error(codes.Internal, "token name cannot be empty")
	}

	// Make sure the role exists. We do not allow people to generate a token for a role that doesn't exist.
	role, err := s.roleStore.GetRole(req.GetRole())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "unable to fetch role %q", req.GetRole())
	}
	if role == nil {
		return nil, status.Errorf(codes.InvalidArgument, "role %q doesn't exist", req.GetRole())
	}

	token, metadata, err := s.backend.IssueRoleToken(req.GetName(), role)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &v1.GenerateTokenResponse{
		Token:    token,
		Metadata: metadata,
	}, nil
}

func (s *serviceImpl) RegisterServiceServer(grpcServer *grpc.Server) {
	v1.RegisterAPITokenServiceServer(grpcServer, s)
}

func (s *serviceImpl) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return v1.RegisterAPITokenServiceHandler(ctx, mux, conn)
}

func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, authorizer.Authorized(ctx, fullMethodName)
}
