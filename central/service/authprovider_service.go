package service

import (
	"fmt"

	"bitbucket.org/stack-rox/apollo/central/db"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/authproviders"
	"bitbucket.org/stack-rox/apollo/pkg/grpc/authz"
	"bitbucket.org/stack-rox/apollo/pkg/grpc/authz/allow"
	"bitbucket.org/stack-rox/apollo/pkg/grpc/authz/service"
	"bitbucket.org/stack-rox/apollo/pkg/grpc/authz/user"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// AuthProviderUpdater knows how to emplace or remove auth providers.
type AuthProviderUpdater interface {
	UpdateProvider(id string, provider authproviders.Authenticator)
	RemoveProvider(id string)
}

// NewAuthProviderService returns the AuthProviderService API.
func NewAuthProviderService(storage db.AuthProviderStorage, auth AuthProviderUpdater) *AuthProviderService {
	return &AuthProviderService{
		storage: storage,
		auth:    auth,
	}
}

// AuthProviderService is the struct that manages the AuthProvider API
type AuthProviderService struct {
	storage db.AuthProviderStorage
	auth    AuthProviderUpdater
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *AuthProviderService) RegisterServiceServer(grpcServer *grpc.Server) {
	v1.RegisterAuthProviderServiceServer(grpcServer, s)
}

// RegisterServiceHandlerFromEndpoint registers this service with the given gRPC Gateway endpoint.
func (s *AuthProviderService) RegisterServiceHandlerFromEndpoint(ctx context.Context, mux *runtime.ServeMux, endpoint string, opts []grpc.DialOption) error {
	return v1.RegisterAuthProviderServiceHandlerFromEndpoint(ctx, mux, endpoint, opts)
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *AuthProviderService) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	pr := service.PerRPC{
		Default: user.Any(),
		Authorizers: map[string]authz.Authorizer{
			"/v1.AuthProviderService/GetAuthProvider":  allow.Anonymous(),
			"/v1.AuthProviderService/GetAuthProviders": allow.Anonymous(),
		},
	}
	return ctx, ReturnErrorCode(pr.Authorized(ctx, fullMethodName))
}

// GetAuthProvider retrieves the authProvider based on the id passed
func (s *AuthProviderService) GetAuthProvider(ctx context.Context, request *v1.ResourceByID) (*v1.AuthProvider, error) {
	if request.GetId() == "" {
		return nil, status.Error(codes.InvalidArgument, "Auth Provider id is required")
	}
	authProvider, exists, err := s.storage.GetAuthProvider(request.GetId())
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, status.Error(codes.NotFound, fmt.Sprintf("Auth Provider %v not found", request.GetId()))
	}
	return authProvider, nil
}

// GetAuthProviders retrieves all authProviders that matches the request filters
func (s *AuthProviderService) GetAuthProviders(ctx context.Context, request *v1.GetAuthProvidersRequest) (*v1.GetAuthProvidersResponse, error) {
	authProviders, err := s.storage.GetAuthProviders(request)
	if err != nil {
		return nil, err
	}
	return &v1.GetAuthProvidersResponse{AuthProviders: authProviders}, nil
}

// PostAuthProvider inserts a new auth provider into the system
func (s *AuthProviderService) PostAuthProvider(ctx context.Context, request *v1.AuthProvider) (*v1.AuthProvider, error) {
	if request == nil {
		return nil, status.Error(codes.InvalidArgument, "request cannot be nil")
	}
	if request.GetId() != "" {
		return nil, status.Error(codes.InvalidArgument, "Auth Provider id must be empty")
	}
	if request.GetLoginUrl() != "" {
		return nil, status.Error(codes.InvalidArgument, "Auth Provider loginUrl field must be empty")
	}
	p, err := authproviders.Create(request)
	if err != nil {
		return nil, err
	}
	request.LoginUrl = p.LoginURL()
	id, err := s.storage.AddAuthProvider(request)
	if err != nil {
		return nil, err
	}
	request.Id = id
	s.auth.UpdateProvider(id, p)
	return request, nil
}

// PutAuthProvider updates an auth provider in the system
func (s *AuthProviderService) PutAuthProvider(ctx context.Context, request *v1.AuthProvider) (*empty.Empty, error) {
	if request == nil {
		return nil, status.Error(codes.InvalidArgument, "request cannot be nil")
	}
	if request.GetId() == "" {
		return nil, status.Error(codes.InvalidArgument, "Auth Provider id is required")
	}

	p, err := authproviders.Create(request)
	if err != nil {
		return nil, err
	}
	request.LoginUrl = p.LoginURL()
	if err := s.storage.UpdateAuthProvider(request); err != nil {
		return nil, err
	}
	s.auth.UpdateProvider(request.GetId(), p)
	return &empty.Empty{}, nil
}

// DeleteAuthProvider deletes an auth provider from the system
func (s *AuthProviderService) DeleteAuthProvider(ctx context.Context, request *v1.ResourceByID) (*empty.Empty, error) {
	if request.GetId() == "" {
		return nil, status.Error(codes.InvalidArgument, "Auth Provider id is required")
	}
	if err := s.storage.RemoveAuthProvider(request.GetId()); err != nil {
		return nil, ReturnErrorCode(err)
	}
	s.auth.RemoveProvider(request.GetId())
	return &empty.Empty{}, nil
}
