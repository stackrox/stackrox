package service

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/auth/userpass"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/authproviders/basic"
	"github.com/stackrox/rox/pkg/errox"
	"google.golang.org/grpc"
)

// basicAuthProviderRemovalServiceImpl is the wrapper around auth provider service which removes access to
// "basic" auth provider type.
type basicAuthProviderRemovalServiceImpl struct {
	underlying Service
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *basicAuthProviderRemovalServiceImpl) RegisterServiceServer(grpcServer *grpc.Server) {
	v1.RegisterAuthProviderServiceServer(grpcServer, s)
}

// RegisterServiceHandler registers this service with the given gRPC Gateway endpoint.
func (s *basicAuthProviderRemovalServiceImpl) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return v1.RegisterAuthProviderServiceHandler(ctx, mux, conn)
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *basicAuthProviderRemovalServiceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, authorizer.Authorized(ctx, fullMethodName)
}

// GetAuthProvider retrieves the authProvider based on the id passed.
func (s *basicAuthProviderRemovalServiceImpl) GetAuthProvider(ctx context.Context, request *v1.GetAuthProviderRequest) (*storage.AuthProvider, error) {
	authProvider, err := s.underlying.GetAuthProvider(ctx, request)
	if err != nil {
		return authProvider, err
	}
	if authProvider.GetType() == basic.TypeName {
		return nil, errors.Wrapf(errox.NotFound, "auth provider %q not found", request.GetId())
	}
	return authProvider, nil
}

// GetLoginAuthProviders retrieves all authProviders that matches the request filters.
func (s *basicAuthProviderRemovalServiceImpl) GetLoginAuthProviders(ctx context.Context, _ *v1.Empty) (*v1.GetLoginAuthProvidersResponse, error) {
	resp, err := s.underlying.GetLoginAuthProviders(ctx, &v1.Empty{})
	if err != nil || len(resp.GetAuthProviders()) == 0 {
		return resp, err
	}
	filtered := make([]*v1.GetLoginAuthProvidersResponse_LoginAuthProvider, 0, len(resp.GetAuthProviders())-1)
	for _, authProvider := range resp.GetAuthProviders() {
		if authProvider.GetType() != basic.TypeName {
			filtered = append(filtered, authProvider)
		}
	}
	return &v1.GetLoginAuthProvidersResponse{
		AuthProviders: filtered,
	}, nil
}

// ListAvailableProviderTypes returns auth provider types which can be created.
func (s *basicAuthProviderRemovalServiceImpl) ListAvailableProviderTypes(ctx context.Context, _ *v1.Empty) (*v1.AvailableProviderTypesResponse, error) {
	return s.underlying.ListAvailableProviderTypes(ctx, &v1.Empty{})
}

// GetAuthProviders retrieves all authProviders that matches the request filters.
func (s *basicAuthProviderRemovalServiceImpl) GetAuthProviders(ctx context.Context, request *v1.GetAuthProvidersRequest) (*v1.GetAuthProvidersResponse, error) {
	resp, err := s.underlying.GetAuthProviders(ctx, request)
	if err != nil {
		return resp, err
	}
	filtered := make([]*storage.AuthProvider, 0, len(resp.GetAuthProviders())-1)
	for _, authProvider := range resp.GetAuthProviders() {
		if authProvider.GetType() != basic.TypeName {
			filtered = append(filtered, authProvider)
		}
	}
	return &v1.GetAuthProvidersResponse{
		AuthProviders: filtered,
	}, nil
}

// PostAuthProvider inserts a new auth provider into the system.
func (s *basicAuthProviderRemovalServiceImpl) PostAuthProvider(ctx context.Context, request *v1.PostAuthProviderRequest) (*storage.AuthProvider, error) {
	if request.GetProvider().GetType() == basic.TypeName {
		return nil, errox.InvalidArgs.Newf("cannot create basic auth providers")
	}
	return s.underlying.PostAuthProvider(ctx, request)
}

func (s *basicAuthProviderRemovalServiceImpl) PutAuthProvider(ctx context.Context, request *storage.AuthProvider) (*storage.AuthProvider, error) {
	if request.GetType() == basic.TypeName {
		return nil, errors.Wrapf(errox.NotFound, "auth provider %q not found", request.GetId())
	}
	return s.underlying.PutAuthProvider(ctx, request)
}

func (s *basicAuthProviderRemovalServiceImpl) UpdateAuthProvider(ctx context.Context, request *v1.UpdateAuthProviderRequest) (*storage.AuthProvider, error) {
	if request.GetId() == userpass.BasicAuthProviderID {
		return nil, errors.Wrapf(errox.NotFound, "auth provider %q not found", request.GetId())
	}
	return s.underlying.UpdateAuthProvider(ctx, request)
}

// DeleteAuthProvider deletes an auth provider from the system.
func (s *basicAuthProviderRemovalServiceImpl) DeleteAuthProvider(ctx context.Context, request *v1.ResourceByID) (*v1.Empty, error) {
	if request.GetId() == userpass.BasicAuthProviderID {
		return nil, errors.Wrapf(errox.NotFound, "auth provider %q not found", request.GetId())
	}
	return s.underlying.DeleteAuthProvider(ctx, request)
}

func (s *basicAuthProviderRemovalServiceImpl) ExchangeToken(ctx context.Context, request *v1.ExchangeTokenRequest) (*v1.ExchangeTokenResponse, error) {
	if request.GetType() == basic.TypeName {
		return nil, errors.Wrapf(errox.NotFound, "auth provider with type %q not found", request.GetType())
	}
	return s.underlying.ExchangeToken(ctx, request)
}
