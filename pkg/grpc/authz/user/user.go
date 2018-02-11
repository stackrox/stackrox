package user

import (
	"context"

	"bitbucket.org/stack-rox/apollo/pkg/grpc/authn"
	"bitbucket.org/stack-rox/apollo/pkg/grpc/authz"
)

type anyUser struct{}

func (anyUser) Authorized(ctx context.Context) error {
	conf, err := authn.FromAuthConfigurationContext(ctx)
	if err != nil {
		return authz.ErrAuthnConfigMissing{}
	}
	// If no authentication provider is configured, default to allow access.
	// This may change in the future, for instance if an administrator login
	// is automatically provisioned.
	if !conf.ProviderConfigured {
		return nil
	}

	identity, err := authn.FromUserContext(ctx)
	if err != nil {
		return authz.ErrNoCredentials{}
	}
	if identity.User.ID == "" {
		return authz.ErrNoCredentials{}
	}
	return nil
}

// Any returns an Authorizer that allows any authenticated user,
// but denies unauthenticated clients.
func Any() authz.Authorizer {
	return anyUser{}
}
