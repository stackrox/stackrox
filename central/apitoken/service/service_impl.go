package service

import (
	"context"
	"fmt"

	"bitbucket.org/stack-rox/apollo/central/apitoken/signer"
	"bitbucket.org/stack-rox/apollo/central/role/resources"
	"bitbucket.org/stack-rox/apollo/central/role/store"
	"bitbucket.org/stack-rox/apollo/central/service"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/auth/permissions"
	"bitbucket.org/stack-rox/apollo/pkg/grpc/authz"
	"bitbucket.org/stack-rox/apollo/pkg/grpc/authz/perrpc"
	"bitbucket.org/stack-rox/apollo/pkg/grpc/authz/user"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		user.With(permissions.Modify(resources.APIToken)): {
			"/v1.APITokenService/GenerateToken",
		},
	})
)

type serviceImpl struct {
	signer    signer.Signer
	roleStore store.Store
}

func (s *serviceImpl) GenerateToken(ctx context.Context, req *v1.GenerateTokenRequest) (*v1.Token, error) {
	// Make sure the role exists. We do not allow people to generate a token for a role that doesn't exist.
	_, exists := s.roleStore.GetRole(req.GetRole())
	if !exists {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("role '%s' doesn't exist", req.GetRole()))
	}
	jwt, err := s.signer.SignedJWT(req.GetRole())
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &v1.Token{Token: jwt}, nil
}

func (s *serviceImpl) RegisterServiceServer(grpcServer *grpc.Server) {
	v1.RegisterAPITokenServiceServer(grpcServer, s)
}

func (s *serviceImpl) RegisterServiceHandlerFromEndpoint(ctx context.Context, mux *runtime.ServeMux, endpoint string, opts []grpc.DialOption) error {
	return v1.RegisterAPITokenServiceHandlerFromEndpoint(ctx, mux, endpoint, opts)
}

func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, service.ReturnErrorCode(authorizer.Authorized(ctx, fullMethodName))
}
