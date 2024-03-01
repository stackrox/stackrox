package service

import (
	"context"
	"sort"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/authprovider/datastore"
	groupDataStore "github.com/stackrox/rox/central/group/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/authproviders"
	"github.com/stackrox/rox/pkg/auth/authproviders/basic"
	"github.com/stackrox/rox/pkg/auth/authproviders/idputil"
	"github.com/stackrox/rox/pkg/auth/permissions"
	userPkg "github.com/stackrox/rox/pkg/auth/user"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/allow"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

var (
	authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		allow.Anonymous(): {
			"/v1.AuthProviderService/ListAvailableProviderTypes",
			// The GetLoginAuthProviders endpoint is used to render the list of providers
			// on the UI login screen. At that point, the user is not authenticated or logged in.
			// Therefore this endpoint should remain anonymously accessible.
			"/v1.AuthProviderService/GetLoginAuthProviders",
			// The ExchangeToken endpoint is used by the UI as part of the
			// user login flow, where user authentication data is not always
			// available. This endpoint should therefore remain
			// anonymous / public.
			"/v1.AuthProviderService/ExchangeToken",
		},
		user.With(permissions.View(resources.Access)): {
			"/v1.AuthProviderService/GetAuthProvider",
			"/v1.AuthProviderService/GetAuthProviders",
		},
		user.With(permissions.Modify(resources.Access)): {
			"/v1.AuthProviderService/PostAuthProvider",
			"/v1.AuthProviderService/UpdateAuthProvider",
			"/v1.AuthProviderService/PutAuthProvider",
			"/v1.AuthProviderService/DeleteAuthProvider",
		},
	})
)

// ClusterService is the struct that manages the cluster API
type serviceImpl struct {
	v1.UnimplementedAuthProviderServiceServer

	registry   authproviders.Registry
	groupStore groupDataStore.DataStore
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

// GetAuthProvider retrieves the authProvider based on the id passed.
func (s *serviceImpl) GetAuthProvider(_ context.Context, request *v1.GetAuthProviderRequest) (*storage.AuthProvider, error) {
	if request.GetId() == "" {
		return nil, errox.InvalidArgs.CausedBy("auth provider id is empty")
	}
	authProvider := s.registry.GetProvider(request.GetId())
	if authProvider == nil {
		return nil, errors.Wrapf(errox.NotFound, "auth provider %q not found", request.GetId())
	}
	return authProvider.StorageView(), nil
}

// GetLoginAuthProviders retrieves all authProviders that matches the request filters.
func (s *serviceImpl) GetLoginAuthProviders(_ context.Context, _ *v1.Empty) (*v1.GetLoginAuthProvidersResponse, error) {
	authProviders := s.registry.GetProviders(nil, nil)
	result := make([]*v1.GetLoginAuthProvidersResponse_LoginAuthProvider, 0, len(authProviders))
	for _, provider := range authProviders {
		if isLoginAuthProvider(provider) {
			view := provider.StorageView()
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

// isLoginAuthProvider is a helper that determines whether an authproviders.Provider can be used for
// login purposes.
func isLoginAuthProvider(provider authproviders.Provider) bool {
	view := provider.StorageView()
	// Only enabled auth providers can be used for login purposes.
	if !view.GetEnabled() {
		return false
	}
	// Providers without a backend factory are useless for login purposes.
	if provider.BackendFactory() == nil {
		return false
	}
	// Only auth providers that are visible should be used for login purposes.
	if view.GetTraits().GetVisibility() != storage.Traits_VISIBLE {
		return false
	}
	return true
}

// ListAvailableProviderTypes returns auth provider types which can be created.
func (s *serviceImpl) ListAvailableProviderTypes(_ context.Context, _ *v1.Empty) (*v1.AvailableProviderTypesResponse, error) {
	factories := s.registry.GetBackendFactories()
	supportedTypes := make([]*v1.AvailableProviderTypesResponse_AuthProviderType, 0, len(factories))
	for typ, factory := range factories {
		if typ == basic.TypeName {
			continue
		}

		attributes := factory.GetSuggestedAttributes()
		sort.Strings(attributes)
		supportedTypes = append(supportedTypes, &v1.AvailableProviderTypesResponse_AuthProviderType{
			Type:                typ,
			SuggestedAttributes: attributes,
		})
	}

	// List auth providers in the same order for consistency across requests.
	sort.Slice(supportedTypes, func(i, j int) bool {
		return supportedTypes[i].GetType() < supportedTypes[j].GetType()
	})

	return &v1.AvailableProviderTypesResponse{
		AuthProviderTypes: supportedTypes,
	}, nil
}

// GetAuthProviders retrieves all authProviders that matches the request filters.
func (s *serviceImpl) GetAuthProviders(_ context.Context, request *v1.GetAuthProvidersRequest) (*v1.GetAuthProvidersResponse, error) {
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

// PostAuthProvider inserts a new auth provider into the system.
func (s *serviceImpl) PostAuthProvider(ctx context.Context, request *v1.PostAuthProviderRequest) (*storage.AuthProvider, error) {
	providerReq := request.GetProvider()
	if providerReq.GetName() == "" {
		return nil, errox.InvalidArgs.CausedBy("no auth provider name specified")
	}
	if providerReq.GetId() != "" {
		return nil, errox.InvalidArgs.CausedBy("auth provider id is not empty")
	}
	if providerReq.GetLoginUrl() != "" {
		return nil, errox.InvalidArgs.CausedBy("auth provider loginUrl field is not empty")
	}
	if providerReq.GetType() == basic.TypeName {
		return nil, errox.InvalidArgs.CausedByf("auth provider of type %s cannot be created",
			basic.TypeName)
	}

	provider, err := s.registry.CreateProvider(ctx, authproviders.WithStorageView(providerReq),
		authproviders.WithValidateCallback(datastore.Singleton()), authproviders.WithAttributeVerifier(providerReq))
	if err != nil {
		return nil, errox.InvalidArgs.New("unable to create an auth provider instance").CausedBy(err)
	}
	return provider.StorageView(), nil
}

// PutAuthProvider upserts an auth provider into the system.
func (s *serviceImpl) PutAuthProvider(ctx context.Context, request *storage.AuthProvider) (*storage.AuthProvider, error) {
	if request.GetId() == "" {
		return nil, errox.InvalidArgs.CausedBy("auth provider id is empty")
	}
	if request.GetType() == basic.TypeName {
		return nil, errox.InvalidArgs.CausedByf("auth provider of type %s cannot be modified",
			basic.TypeName)
	}

	provider := s.registry.GetProvider(request.GetId())
	if provider == nil {
		return nil, errox.NotFound.Newf("auth provider with id %q does not exist", request.GetId())
	}

	// Attempt to merge configs.
	request.Config = provider.MergeConfigInto(request.GetConfig())

	if err := s.registry.ValidateProvider(ctx, authproviders.WithStorageView(request)); err != nil {
		return nil, errox.InvalidArgs.New("auth provider validation check failed").CausedBy(err)
	}

	// This will not log anyone out as the provider was not validated and thus no one has ever logged into it.
	if err := s.registry.DeleteProvider(ctx, request.GetId(), false, false); err != nil {
		return nil, err
	}

	provider, err := s.registry.CreateProvider(ctx, authproviders.WithStorageView(request),
		authproviders.WithAttributeVerifier(request),
		authproviders.WithValidateCallback(datastore.Singleton()))
	if err != nil {
		return nil, errox.InvalidArgs.New("unable to create an auth provider instance").CausedBy(err)
	}
	return provider.StorageView(), nil
}

// UpdateAuthProvider updates an auth provider within the system.
func (s *serviceImpl) UpdateAuthProvider(ctx context.Context, request *v1.UpdateAuthProviderRequest) (*storage.AuthProvider, error) {
	if request.GetId() == "" {
		return nil, errox.InvalidArgs.CausedBy("auth provider id is empty")
	}

	// Get auth provider.
	authProvider := s.registry.GetProvider(request.GetId())
	if authProvider == nil {
		return nil, errox.NotFound.CausedByf("auth provider %q not found", request.GetId())
	}

	// Do not attempt to update auth provider of type "basic" and instead return an invalid args error.
	if authProvider.StorageView().GetType() == basic.TypeName {
		return nil, errox.InvalidArgs.CausedByf("auth provider of type %s cannot be modified",
			basic.TypeName)
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
		return nil, errox.InvalidArgs.New("unable to update auth provider").CausedBy(err)
	}
	return provider.StorageView(), nil
}

// DeleteAuthProvider deletes an auth provider from the system
func (s *serviceImpl) DeleteAuthProvider(ctx context.Context, request *v1.DeleteByIDWithForce) (*v1.Empty, error) {
	if request.GetId() == "" {
		return nil, errox.InvalidArgs.CausedBy("auth provider id is empty")
	}
	// Get auth provider.
	authProvider := s.registry.GetProvider(request.GetId())
	if authProvider == nil {
		return nil, errors.Wrapf(errox.NotFound, "auth provider %q not found", request.GetId())
	}

	// Do not attempt to delete auth provider of type "basic" and instead return an invalid args error.
	if authProvider.StorageView().GetType() == basic.TypeName {
		return nil, errox.InvalidArgs.CausedByf("auth provider of type %s cannot be deleted",
			basic.TypeName)
	}

	// Delete auth provider.
	if err := s.registry.DeleteProvider(ctx, request.GetId(), request.GetForce(), true); err != nil {
		return nil, err
	}
	// Delete groups for auth provider.
	if err := s.groupStore.RemoveAllWithAuthProviderID(ctx, request.GetId(), request.GetForce()); err != nil {
		return nil, errors.Wrapf(err, "failed to delete groups associated with auth provider %q", request.GetId())
	}
	return &v1.Empty{}, nil
}

// ExchangeToken exchanges a token from an auth provider from the system.
func (s *serviceImpl) ExchangeToken(ctx context.Context, request *v1.ExchangeTokenRequest) (*v1.ExchangeTokenResponse, error) {
	provider, err := s.registry.ResolveProvider(request.GetType(), request.GetState())
	if err != nil {
		return nil, err
	}

	authResponse, clientState, err := s.registry.GetExternalUserClaim(ctx, request.GetExternalToken(), request.GetType(), request.GetState())
	if err != nil {
		return nil, err
	}

	clientState, mode := idputil.ParseClientState(clientState)
	testMode := mode == idputil.TestAuthMode
	response := &v1.ExchangeTokenResponse{
		ClientState: clientState,
		Test:        testMode,
	}

	userMetadata, err := authproviders.CreateRoleBasedIdentity(sac.WithAllAccess(ctx), provider, authResponse)
	if err != nil {
		if testMode {
			return nil, errors.Wrap(err, "cannot create role based identity")
		}
		log.Warnf("Error creating role based identity: %v", err)
	}
	userPkg.LogSuccessfulUserLogin(log, userMetadata)

	if testMode {
		response.User = userMetadata
		return response, nil
	}

	token, refreshCookie, err := s.registry.IssueToken(ctx, provider, authResponse)
	if err != nil {
		return nil, err
	}
	response.Token = token.Token
	if refreshCookie != nil {
		if err := grpc.SetHeader(ctx, metadata.Pairs("Set-Cookie", refreshCookie.String())); err != nil {
			log.Errorf("Failed to set cookie in gRPC response: %v", err)
		}
	}
	accessTokenCookie := authproviders.AccessTokenCookie(token)
	if err := grpc.SetHeader(ctx, metadata.Pairs("Set-Cookie", accessTokenCookie.String())); err != nil {
		return nil, errox.NoCredentials.CausedBy(err)
	}

	return response, nil
}
