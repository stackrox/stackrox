package service

import (
	"fmt"

	"bitbucket.org/stack-rox/apollo/central/authprovider/store"
	"bitbucket.org/stack-rox/apollo/central/service"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/authproviders"
	"bitbucket.org/stack-rox/apollo/pkg/grpc/authz"
	"bitbucket.org/stack-rox/apollo/pkg/grpc/authz/allow"
	authService "bitbucket.org/stack-rox/apollo/pkg/grpc/authz/service"
	"bitbucket.org/stack-rox/apollo/pkg/grpc/authz/user"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ClusterService is the struct that manages the cluster API
type serviceImpl struct {
	storage store.Store
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *serviceImpl) RegisterServiceServer(grpcServer *grpc.Server) {
	v1.RegisterAuthProviderServiceServer(grpcServer, s)
}

// RegisterServiceHandlerFromEndpoint registers this service with the given gRPC Gateway endpoint.
func (s *serviceImpl) RegisterServiceHandlerFromEndpoint(ctx context.Context, mux *runtime.ServeMux, endpoint string, opts []grpc.DialOption) error {
	return v1.RegisterAuthProviderServiceHandlerFromEndpoint(ctx, mux, endpoint, opts)
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	pr := authService.PerRPC{
		Default: user.Any(),
		Authorizers: map[string]authz.Authorizer{
			"/v1.AuthProviderService/GetAuthProvider":  allow.Anonymous(),
			"/v1.AuthProviderService/GetAuthProviders": allow.Anonymous(),
		},
	}
	return ctx, service.ReturnErrorCode(pr.Authorized(ctx, fullMethodName))
}

// GetAuthProvider retrieves the authProvider based on the id passed
func (s *serviceImpl) GetAuthProvider(ctx context.Context, request *v1.ResourceByID) (*v1.AuthProvider, error) {
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
func (s *serviceImpl) GetAuthProviders(ctx context.Context, request *v1.GetAuthProvidersRequest) (*v1.GetAuthProvidersResponse, error) {
	authProviders, err := s.storage.GetAuthProviders(request)
	if err != nil {
		return nil, err
	}
	return &v1.GetAuthProvidersResponse{AuthProviders: authProviders}, nil
}

// PostAuthProvider inserts a new auth provider into the system
func (s *serviceImpl) PostAuthProvider(ctx context.Context, request *v1.AuthProvider) (*v1.AuthProvider, error) {
	if request == nil {
		return nil, status.Error(codes.InvalidArgument, "request cannot be nil")
	}
	if request.GetId() != "" {
		return nil, status.Error(codes.InvalidArgument, "Auth Provider id must be empty")
	}
	if request.GetLoginUrl() != "" {
		return nil, status.Error(codes.InvalidArgument, "Auth Provider loginUrl field must be empty")
	}
	loginURL, err := authproviders.LoginURLFromProto(request)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	request.LoginUrl = loginURL
	id, err := s.storage.AddAuthProvider(request)
	if err != nil {
		return nil, err
	}
	request.Id = id
	return request, nil
}

// PutAuthProvider updates an auth provider in the system
func (s *serviceImpl) PutAuthProvider(ctx context.Context, request *v1.AuthProvider) (*empty.Empty, error) {
	if request == nil {
		return nil, status.Error(codes.InvalidArgument, "request cannot be nil")
	}
	if request.GetId() == "" {
		return nil, status.Error(codes.InvalidArgument, "Auth Provider id is required")
	}

	loginURL, err := authproviders.LoginURLFromProto(request)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	request.LoginUrl = loginURL
	if err := s.storage.UpdateAuthProvider(request); err != nil {
		return nil, err
	}
	return &empty.Empty{}, nil
}

// DeleteAuthProvider deletes an auth provider from the system
func (s *serviceImpl) DeleteAuthProvider(ctx context.Context, request *v1.ResourceByID) (*empty.Empty, error) {
	if request.GetId() == "" {
		return nil, status.Error(codes.InvalidArgument, "Auth Provider id is required")
	}
	if err := s.storage.RemoveAuthProvider(request.GetId()); err != nil {
		return nil, service.ReturnErrorCode(err)
	}
	return &empty.Empty{}, nil
}
