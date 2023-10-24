package service

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"time"

	"github.com/gogo/protobuf/types"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	pkgErrors "github.com/pkg/errors"
	"github.com/stackrox/rox/central/auth/datastore"
	"github.com/stackrox/rox/central/convert/storagetov1"
	"github.com/stackrox/rox/central/convert/v1tostorage"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	userPkg "github.com/stackrox/rox/pkg/auth/user"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/allow"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/uuid"
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
			// is present, it will return HTTP 4xx status code.
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

	errInvalidTokenExpiration   = errors.New("invalid token expiration duration provided")
	errInvalidIssuer            = errors.New("invalid token issuer provided")
	errInvalidRegularExpression = errors.New("invalid regular expression provided")
)

type serviceImpl struct {
	authDataStore datastore.DataStore

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
		return nil, pkgErrors.Errorf("expiration time: %s", err)
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

func (s *serviceImpl) ListAuthMachineToMachineConfigs(ctx context.Context, _ *v1.Empty) (*v1.ListAuthMachineToMachineConfigResponse, error) {
	storageConfigs, err := s.authDataStore.ListAuthM2MConfigs(ctx)
	if err != nil {
		return nil, err
	}

	return &v1.ListAuthMachineToMachineConfigResponse{Configs: storagetov1.AuthM2MConfigs(storageConfigs)}, nil
}

func (s *serviceImpl) GetAuthMachineToMachineConfig(ctx context.Context, id *v1.ResourceByID) (*v1.GetAuthMachineToMachineConfigResponse, error) {
	config, exists, err := s.authDataStore.GetAuthM2MConfig(ctx, id.GetId())
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errox.NotFound.Newf("auth machine to machine config with id %q", id.GetId())
	}
	return &v1.GetAuthMachineToMachineConfigResponse{Config: storagetov1.AuthM2MConfig(config)}, nil
}

func (s *serviceImpl) AddAuthMachineToMachineConfig(ctx context.Context, request *v1.AddAuthMachineToMachineConfigRequest) (*v1.AddAuthMachineToMachineConfigResponse, error) {
	config := request.GetConfig()
	if err := s.validateAuthMachineToMachineConfig(config, true); err != nil {
		return nil, err
	}
	config.Id = uuid.NewV4().String()
	storageConfig, err := s.authDataStore.AddAuthM2MConfig(ctx, v1tostorage.AuthM2MConfig(config))
	if err != nil {
		return nil, err
	}

	return &v1.AddAuthMachineToMachineConfigResponse{Config: storagetov1.AuthM2MConfig(storageConfig)}, nil
}

func (s *serviceImpl) UpdateAuthMachineToMachineConfig(ctx context.Context, request *v1.UpdateAuthMachineToMachineConfigRequest) (*v1.Empty, error) {
	config := request.GetConfig()
	if err := s.validateAuthMachineToMachineConfig(config, false); err != nil {
		return nil, err
	}

	if err := s.authDataStore.UpdateAuthM2MConfig(ctx, v1tostorage.AuthM2MConfig(config)); err != nil {
		return nil, err
	}

	return &v1.Empty{}, nil
}

func (s *serviceImpl) DeleteAuthMachineToMachineConfig(ctx context.Context, id *v1.ResourceByID) (*v1.Empty, error) {
	if err := s.authDataStore.RemoveAuthM2MConfig(ctx, id.GetId()); err != nil {
		return nil, errox.InvalidArgs.
			Newf("could not delete auth machine to machine config with id %q", id.GetId()).CausedBy(err)
	}
	return &v1.Empty{}, nil
}

func (s *serviceImpl) ExchangeAuthMachineToMachineToken(_ context.Context, _ *v1.ExchangeAuthMachineToMachineTokenRequest) (*v1.ExchangeAuthMachineToMachineTokenResponse, error) {
	return nil, errox.InvariantViolation.New("not yet implemented")
}

func (s *serviceImpl) validateAuthMachineToMachineConfig(config *v1.AuthMachineToMachineConfig, skipIDCheck bool) error {
	if config == nil {
		return errox.InvalidArgs.New("empty config given")
	}
	if config.GetId() == "" && !skipIDCheck {
		return errox.InvalidArgs.New("empty ID given")
	}

	duration, err := time.ParseDuration(config.GetTokenExpirationDuration())
	if err != nil {
		return fmt.Errorf("%w: %w: %w", errox.InvalidArgs, errInvalidTokenExpiration, err)
	}

	if duration < time.Minute || duration > 24*time.Hour {
		return fmt.Errorf("%w: %w: token expiration must be between 1 minute and 24 hours, but was %s",
			errox.InvalidArgs, errInvalidTokenExpiration, duration.String())
	}

	if config.GetType() == v1.AuthMachineToMachineConfig_GENERIC && config.GetIssuer() == "" {
		return fmt.Errorf("%w: %w: type %s was used, but no configuration for the issuer was given",
			errox.InvalidArgs, errInvalidIssuer, storage.AuthMachineToMachineConfig_GENERIC)
	}

	if config.GetIssuer() != "" {
		parsedIssuer, err := url.Parse(config.GetIssuer())
		if err != nil {
			return fmt.Errorf("%w: %w: %w", errox.InvalidArgs, errInvalidIssuer, err)
		}
		if parsedIssuer.Scheme == "http" {
			return fmt.Errorf("%w: %w: HTTPS is required for the issuer", errox.InvalidArgs, errInvalidIssuer)
		}
	}

	var regexValidationErrs error
	for _, mapping := range config.GetMappings() {
		if _, err := regexp.Compile(mapping.GetValueExpression()); err != nil {
			regexValidationErrs = errors.Join(regexValidationErrs,
				fmt.Errorf("%w for key %q: %w", errInvalidRegularExpression, mapping.GetKey(), err))
		}
	}

	if regexValidationErrs != nil {
		return fmt.Errorf("%w: %w", errox.InvalidArgs, regexValidationErrs)
	}

	return nil
}
