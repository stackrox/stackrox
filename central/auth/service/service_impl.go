package service

import (
	"context"

	"github.com/gogo/protobuf/types"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/auth/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/auth/permissions"
	userPkg "github.com/stackrox/rox/pkg/auth/user"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/allow"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"github.com/stackrox/rox/pkg/sac/resources"
	"google.golang.org/grpc"
)

var (
	_ v1.AuthServiceServer = (*serviceImpl)(nil)

	authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		allow.Anonymous(): {
			// GetAuthStatus does return information about the caller identity
			// (present in context). In case it is called by an anonymous
			// user, it will return HTTP 401 (unauthorised) which is
			// semantically correct.
			"/v1.AuthService/GetAuthStatus",
			// ExchangeAuthM2MToken exchanges an identity token of a third-party
			// OIDC provider with a Central access token, and hence needs to allow
			// calls by anonymous users. In case no config for exchanging the token
			// is present, it will return HTTP 401.
			"/v1.AuthService/ExchangeAuthMachineToMachineToken",
		},
		user.With(permissions.View(resources.Access)): {
			"/v1.AuthService/ListAuthMachineToMachineConfigs",
			"/v1.AuthService/GetAuthMachineToMachineConfig",
		},
		user.With(permissions.Modify(resources.Access)): {
			"/v1.AuthService/DeleteAuthMachineToMachineConfig",
			"/v1.AuthService/AddAuthMachineToMachineConfig",
			"/v1.AuthService/UpdateAuthMachineToMachineConfig",
		},
	})
)

type serviceImpl struct {
	ds datastore.DataStore

	v1.UnimplementedAuthServiceServer
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *serviceImpl) RegisterServiceServer(grpcServer *grpc.Server) {
	v1.RegisterAuthServiceServer(grpcServer, s)
}

// RegisterServiceHandler registers this service with the given gRPC Gateway endpoint.
func (s *serviceImpl) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return v1.RegisterAuthServiceHandler(ctx, mux, conn)
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, authorizer.Authorized(ctx, fullMethodName)
}

// GetAuthStatus retrieves the auth status based on the credentials given to the server.
func (s *serviceImpl) GetAuthStatus(ctx context.Context, _ *v1.Empty) (*v1.AuthStatus, error) {
	id, err := authn.IdentityFromContext(ctx)
	if err != nil {
		return nil, err
	}

	return authStatusForID(id)
}

func authStatusForID(id authn.Identity) (*v1.AuthStatus, error) {
	_, notValidAfter := id.ValidityPeriod()
	exp, err := types.TimestampProto(notValidAfter)
	if err != nil {
		return nil, errors.Errorf("expiration time: %s", err)
	}

	result := &v1.AuthStatus{
		Expires:        exp,
		UserInfo:       id.User().Clone(),
		UserAttributes: userPkg.ConvertAttributes(id.Attributes()),
	}

	if provider := id.ExternalAuthProvider(); provider != nil {
		// every Identity should now have an auth provider but API token Identities won't have a Backend
		if backend := provider.Backend(); backend != nil {
			result.RefreshUrl = backend.RefreshURL()
		}
		authProvider := provider.StorageView().Clone()
		if authProvider != nil {
			// config might contain semi-sensitive values, so strip it
			authProvider.Config = nil
		}
		result.AuthProvider = authProvider
	}
	if svc := id.Service(); svc != nil {
		result.Id = &v1.AuthStatus_ServiceId{ServiceId: svc}
	} else {
		result.Id = &v1.AuthStatus_UserId{UserId: id.UID()}
	}
	return result, nil
}

func (s *serviceImpl) ListAuthM2MConfigs(ctx context.Context, _ *v1.Empty) (*v1.ListAuthMachineToMachineConfigResponse, error) {
	storageConfigs, err := s.ds.ListAuthM2MConfigs(ctx)
	if err != nil {
		return nil, err
	}

	return &v1.ListAuthMachineToMachineConfigResponse{Configs: toV1Protos(storageConfigs)}, nil
}

func (s *serviceImpl) GetAuthM2MConfig(ctx context.Context, id *v1.ResourceByID) (*v1.GetAuthMachineToMachineConfigResponse, error) {
	config, exists, err := s.ds.GetAuthM2MConfig(ctx, id.GetId())
	if !exists {
		return nil, errox.NotFound.Newf("auth machine to machine config with id %q", id.GetId())
	}
	if err != nil {
		return nil, err
	}
	return &v1.GetAuthMachineToMachineConfigResponse{Config: toV1Proto(config)}, nil
}

func (s *serviceImpl) PostAuthM2MConfig(ctx context.Context, request *v1.AddAuthMachineToMachineConfigRequest) (*v1.AddAuthMachineToMachineConfigResponse, error) {
	storageConfig, err := s.ds.AddAuthM2MConfig(ctx, toStorageProto(request.GetConfig()))
	if err != nil {
		return nil, err
	}

	return &v1.AddAuthMachineToMachineConfigResponse{Config: toV1Proto(storageConfig)}, nil
}

func (s *serviceImpl) DeleteAuthM2MConfig(ctx context.Context, id *v1.ResourceByID) (*v1.Empty, error) {
	if err := s.ds.RemoveAuthM2MConfig(ctx, id.GetId()); err != nil {
		return nil, err
	}
	return &v1.Empty{}, nil
}

func (s *serviceImpl) ExchangeAuthM2MToken(_ context.Context, _ *v1.ExchangeAuthMachineToMachineTokenRequest) (*v1.ExchangeAuthMachineToMachineTokenResponse, error) {
	return nil, errox.InvariantViolation.New("not yet implemented")
}
