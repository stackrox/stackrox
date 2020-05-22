package service

import (
	"context"
	"fmt"
	"sort"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/authprovider/datastore"
	"github.com/stackrox/rox/central/role/resources"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/authproviders"
	"github.com/stackrox/rox/pkg/auth/authproviders/basic"
	"github.com/stackrox/rox/pkg/auth/authproviders/idputil"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/allow"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"github.com/stackrox/rox/pkg/sac"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		allow.Anonymous(): {
			"/v1.AuthProviderService/GetLoginAuthProviders",
			"/v1.AuthProviderService/ExchangeToken",
		},
		user.With(permissions.View(resources.AuthProvider)): {
			"/v1.AuthProviderService/GetAuthProvider",
			"/v1.AuthProviderService/GetAuthProviders",
		},
		user.With(permissions.Modify(resources.AuthProvider)): {
			"/v1.AuthProviderService/PostAuthProvider",
			"/v1.AuthProviderService/UpdateAuthProvider",
			"/v1.AuthProviderService/PutAuthProvider",
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
func (s *serviceImpl) GetAuthProvider(ctx context.Context, request *v1.GetAuthProviderRequest) (*storage.AuthProvider, error) {
	if request.GetId() == "" {
		return nil, status.Error(codes.InvalidArgument, "Auth Provider id is required")
	}
	authProvider := s.registry.GetProvider(request.GetId())
	if authProvider == nil {
		return nil, status.Error(codes.NotFound, fmt.Sprintf("Auth Provider %v not found", request.GetId()))
	}
	return authProvider.StorageView(), nil
}

// GetLoginAuthProviders retrieves all authProviders that matches the request filters
func (s *serviceImpl) GetLoginAuthProviders(ctx context.Context, empty *v1.Empty) (*v1.GetLoginAuthProvidersResponse, error) {
	authProviders := s.registry.GetProviders(nil, nil)
	result := make([]*v1.GetLoginAuthProvidersResponse_LoginAuthProvider, 0, len(authProviders))
	for _, provider := range authProviders {
		if view := provider.StorageView(); view.GetEnabled() {
			result = append(result, &v1.GetLoginAuthProvidersResponse_LoginAuthProvider{
				Id:       view.GetId(),
				Name:     view.GetName(),
				Type:     view.GetType(),
				LoginUrl: view.GetLoginUrl(),
			})
		}
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].GetType() == basic.TypeName {
			return false
		}
		if result[j].GetType() == basic.TypeName {
			return true
		}
		return result[i].GetName() < result[j].GetName()
	})
	return &v1.GetLoginAuthProvidersResponse{AuthProviders: result}, nil
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

	authProviders := s.registry.GetProviders(name, typ)
	result := make([]*storage.AuthProvider, len(authProviders))
	for i, provider := range authProviders {
		result[i] = provider.StorageView()
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].GetType() == basic.TypeName {
			return false
		}
		if result[j].GetType() == basic.TypeName {
			return true
		}
		return result[i].GetName() < result[j].GetName()
	})
	return &v1.GetAuthProvidersResponse{AuthProviders: result}, nil
}

// PostAuthProvider inserts a new auth provider into the system
func (s *serviceImpl) PostAuthProvider(ctx context.Context, request *v1.PostAuthProviderRequest) (*storage.AuthProvider, error) {
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

	provider, err := s.registry.CreateProvider(ctx, authproviders.WithStorageView(providerReq), authproviders.WithValidateCallback(datastore.Singleton()))
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "could not create auth provider: %v", err)
	}
	return provider.StorageView(), nil
}

func (s *serviceImpl) PutAuthProvider(ctx context.Context, request *storage.AuthProvider) (*storage.AuthProvider, error) {
	if request.GetId() == "" {
		return nil, status.Error(codes.InvalidArgument, "Auth Provider id must not be empty")
	}

	provider := s.registry.GetProvider(request.GetId())
	if provider == nil {
		return nil, status.Errorf(codes.InvalidArgument, "Provider with id %q does not exist", request.GetId())
	}

	// Attempt to merge configs.
	request.Config = provider.MergeConfigInto(request.GetConfig())

	// This will not log anyone out as the provider was not validated and thus no one has ever logged into it
	if err := s.registry.DeleteProvider(ctx, request.GetId(), false); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	provider, err := s.registry.CreateProvider(ctx, authproviders.WithStorageView(request), authproviders.WithValidateCallback(datastore.Singleton()))
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "could not create auth provider: %v", err)
	}
	return provider.StorageView(), nil
}

func (s *serviceImpl) UpdateAuthProvider(ctx context.Context, request *v1.UpdateAuthProviderRequest) (*storage.AuthProvider, error) {
	if request.GetId() == "" {
		return nil, status.Error(codes.InvalidArgument, "Auth Provider id must not be empty")
	}

	var options []authproviders.ProviderOption
	if nameOpt, ok := request.GetNameOpt().(*v1.UpdateAuthProviderRequest_Name); ok {
		options = append(options, authproviders.WithName(nameOpt.Name))
	}
	if enabledOpt, ok := request.GetEnabledOpt().(*v1.UpdateAuthProviderRequest_Enabled); ok {
		options = append(options, authproviders.WithEnabled(enabledOpt.Enabled))
	}
	provider, err := s.registry.UpdateProvider(ctx, request.GetId(), options...)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "could not update auth provider: %v", err)
	}
	return provider.StorageView(), nil
}

// DeleteAuthProvider deletes an auth provider from the system
func (s *serviceImpl) DeleteAuthProvider(ctx context.Context, request *v1.ResourceByID) (*v1.Empty, error) {
	if request.GetId() == "" {
		return nil, status.Error(codes.InvalidArgument, "Auth Provider id is required")
	}
	if err := s.registry.DeleteProvider(ctx, request.GetId(), true); err != nil {
		return nil, err
	}
	return &v1.Empty{}, nil
}

func (s *serviceImpl) ExchangeToken(ctx context.Context, request *v1.ExchangeTokenRequest) (*v1.ExchangeTokenResponse, error) {
	provider, err := s.registry.ResolveProvider(request.GetType(), request.GetState())
	if err != nil {
		return nil, err
	}

	authResponse, clientState, err := s.registry.GetExternalUserClaim(ctx, request.GetExternalToken(), request.GetType(), request.GetState())
	if err != nil {
		return nil, err
	}

	clientState, testMode := idputil.ParseClientState(clientState)
	response := &v1.ExchangeTokenResponse{
		ClientState: clientState,
		Test:        testMode,
	}

	if testMode {
		// We need all access for retrieving roles.
		userMetadata, err := authproviders.CreateRoleBasedIdentity(sac.WithAllAccess(ctx), provider, authResponse)
		if err != nil {
			return nil, errors.Wrap(err, "cannot create role based identity")
		}
		response.User = userMetadata
		return response, nil
	}

	// We need all access for retrieving roles.
	token, err := s.registry.IssueToken(ctx, provider, authResponse)
	if err != nil {
		return nil, err
	}
	response.Token = token
	return response, nil
}
