package user

import (
	"context"
	"fmt"

	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/grpc/authz"
)

type permissionChecker struct {
	requiredPermissions []permissions.Permission
}

func (p *permissionChecker) Authorized(ctx context.Context, _ string) error {
	conf, err := authn.FromAuthConfigurationContext(ctx)
	if err != nil {
		return authz.ErrAuthnConfigMissing
	}
	// If no authentication provider is configured, default to allow access.
	// This may change in the future, for instance if an administrator login
	// is automatically provisioned.
	if !conf.ProviderConfigured {
		return nil
	}

	identity, err := authn.FromTokenBasedIdentityContext(ctx)
	if err != nil {
		return authz.ErrNoCredentials
	}
	if identity.ID() == "" || identity.Role() == nil {
		return authz.ErrNoCredentials
	}
	for _, permission := range p.requiredPermissions {
		if !identity.Role().Has(permission) {
			return authz.ErrNotAuthorized(fmt.Sprintf("not authorized to %s %s",
				permission.Access, permission.Resource))
		}
	}
	return nil
}

// With returns an authorizer that only authorizes users/tokens
// which satisfy all the given permissions.
func With(requiredPermissions ...permissions.Permission) authz.Authorizer {
	return &permissionChecker{requiredPermissions}
}
