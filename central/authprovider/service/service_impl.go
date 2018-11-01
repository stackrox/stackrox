package service

import (
	"fmt"
	"sort"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/auth/authproviders"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/allow"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		allow.Anonymous(): {
			"/v1.AuthProviderService/GetAuthProvider",
			"/v1.AuthProviderService/GetAuthProviders",
			"/v1.AuthProviderService/ExchangeToken",
		},
		user.With(permissions.Modify(resources.AuthProvider)): {
			"/v1.AuthProviderService/PostAuthProvider",
			"/v1.AuthProviderService/UpdateAuthProvider",
			"/v1.AuthProviderService/DeleteAuthProvider",
		},
	})
)

// ClusterService is the struct that manages the cluster API
type serviceImpl struct {
	registry authproviders.Registry
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *serviceImpl) RegisterServiceServer(grpcServer *grpc.Server) {
	v1.RegisterAuthProviderServiceServer(grpcServer, s)
}

// RegisterServiceHandler registers this service with the given gRPC Gateway endpoint.
func (s *serviceImpl) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return v1.RegisterAuthProviderServiceHandler(ctx, mux, conn)
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, authorizer.Authorized(ctx, fullMethodName)
}

// GetAuthProvider retrieves the authProvider based on the id passed
func (s *serviceImpl) GetAuthProvider(ctx context.Context, request *v1.GetAuthProviderRequest) (*v1.AuthProvider, error) {
	if request.GetId() == "" {
		return nil, status.Error(codes.InvalidArgument, "Auth Provider id is required")
	}
	authProvider := s.registry.GetAuthProvider(ctx, request.GetId())
	if authProvider == nil {
		return nil, status.Error(codes.NotFound, fmt.Sprintf("Auth Provider %v not found", request.GetId()))
	}
	return authProvider.AsV1(), nil
}

// GetAuthProviders retrieves all authProviders that matches the request filters
func (s *serviceImpl) GetAuthProviders(ctx context.Context, request *v1.GetAuthProvidersRequest) (*v1.GetAuthProvidersResponse, error) {
	var name, typ *string
	if request.GetName() != "" {
		name = &request.Name
	}
	if request.GetType() != "" {
		typ = &request.Type
	}

	authProviders := s.registry.GetAuthProviders(ctx, name, typ)
	result := make([]*v1.AuthProvider, len(authProviders))
	for i, provider := range authProviders {
		result[i] = provider.AsV1()
	}
	sort.SliceStable(result, func(i, j int) bool {
		return result[i].GetName() < result[j].GetName()
	})
	return &v1.GetAuthProvidersResponse{AuthProviders: result}, nil
}

// PostAuthProvider inserts a new auth provider into the system
func (s *serviceImpl) PostAuthProvider(ctx context.Context, request *v1.PostAuthProviderRequest) (*v1.AuthProvider, error) {
	providerReq := request.GetProvider()
	if providerReq.GetName() == "" {
		return nil, status.Error(codes.InvalidArgument, "no provider name specified")
	}
	if providerReq.GetId() != "" {
		return nil, status.Error(codes.InvalidArgument, "Auth Provider id must be empty")
	}
	if providerReq.GetLoginUrl() != "" {
		return nil, status.Error(codes.InvalidArgument, "Auth Provider loginUrl field must be empty")
	}
	provider, err := s.registry.CreateAuthProvider(ctx, providerReq.GetType(), providerReq.GetName(), providerReq.GetUiEndpoint(), providerReq.GetEnabled(), providerReq.GetValidated(), providerReq.GetConfig())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "could not create auth provider: %v", err)
	}
	return provider.AsV1(), nil
}

func (s *serviceImpl) UpdateAuthProvider(ctx context.Context, request *v1.UpdateAuthProviderRequest) (*v1.AuthProvider, error) {
	if request.GetId() == "" {
		return nil, status.Error(codes.InvalidArgument, "Auth Provider id must not be empty")
	}
	var name *string
	var enabled *bool
	if nameOpt, ok := request.GetNameOpt().(*v1.UpdateAuthProviderRequest_Name); ok {
		name = &nameOpt.Name
	}
	if enabledOpt, ok := request.GetEnabledOpt().(*v1.UpdateAuthProviderRequest_Enabled); ok {
		enabled = &enabledOpt.Enabled
	}
	provider, err := s.registry.UpdateAuthProvider(ctx, request.GetId(), name, enabled)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "could not update auth provider: %v", err)
	}
	return provider.AsV1(), nil
}

// DeleteAuthProvider deletes an auth provider from the system
func (s *serviceImpl) DeleteAuthProvider(ctx context.Context, request *v1.ResourceByID) (*v1.Empty, error) {
	if request.GetId() == "" {
		return nil, status.Error(codes.InvalidArgument, "Auth Provider id is required")
	}
	if err := s.registry.DeleteAuthProvider(ctx, request.GetId()); err != nil {
		return nil, err
	}
	return &v1.Empty{}, nil
}

func (s *serviceImpl) ExchangeToken(ctx context.Context, request *v1.ExchangeTokenRequest) (*v1.ExchangeTokenResponse, error) {
	token, clientState, err := s.registry.ExchangeToken(ctx, request.GetExternalToken(), request.GetType(), request.GetState())
	if err != nil {
		return nil, err
	}
	return &v1.ExchangeTokenResponse{
		Token:       token,
		ClientState: clientState,
	}, nil
}
