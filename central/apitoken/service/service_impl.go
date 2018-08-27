package service

import (
	"context"
	"fmt"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/stackrox/rox/central/apitoken/cachedstore"
	"github.com/stackrox/rox/central/apitoken/parser"
	"github.com/stackrox/rox/central/apitoken/signer"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/central/role/store"
	"github.com/stackrox/rox/central/service"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"github.com/stackrox/rox/pkg/protoconv"
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
	signer     signer.Signer
	parser     parser.Parser
	roleStore  store.Store
	tokenStore cachedstore.CachedStore
}

func (s *serviceImpl) GetAPIToken(ctx context.Context, req *v1.ResourceByID) (*v1.TokenMetadata, error) {
	if req.GetId() == "" {
		return nil, status.Error(codes.InvalidArgument, "empty id passed")
	}
	token, exists, err := s.tokenStore.GetToken(req.GetId())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "token retrieval failed: %s", err)
	}
	if !exists {
		return nil, status.Errorf(codes.InvalidArgument, "token with id '%s' does not exist", req.GetId())
	}
	return token, nil
}

func (s *serviceImpl) GetAPITokens(ctx context.Context, req *v1.GetAPITokensRequest) (*v1.GetAPITokensResponse, error) {
	tokens, err := s.tokenStore.GetTokens(req)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "retrieval of tokens failed: %s", err)
	}
	return &v1.GetAPITokensResponse{
		Tokens: tokens,
	}, nil
}

func (s *serviceImpl) RevokeToken(ctx context.Context, req *v1.ResourceByID) (*v1.Empty, error) {
	exists, err := s.tokenStore.RevokeToken(req.GetId())
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
	_, exists := s.roleStore.GetRole(req.GetRole())
	if !exists {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("role '%s' doesn't exist", req.GetRole()))
	}

	token, id, issuedAt, expiration, err := s.signer.SignedJWT(req.GetRole())
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	metadata := &v1.TokenMetadata{
		Id:         id,
		Name:       req.GetName(),
		Role:       req.GetRole(),
		IssuedAt:   protoconv.ConvertTimeToTimestamp(issuedAt),
		Expiration: protoconv.ConvertTimeToTimestamp(expiration),
	}
	err = s.tokenStore.AddToken(metadata)
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
	return ctx, service.ReturnErrorCode(authorizer.Authorized(ctx, fullMethodName))
}
