package user

import (
	"context"
	"fmt"

	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/grpc/authz"
)

type permissionChecker struct {
	requiredPermissions []*v1.Permission
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

	id := authn.IdentityFromContext(ctx)
	if id == nil {
		return authz.ErrNoCredentials
	}
	role := id.Role()
	if role == nil {
		return authz.ErrNoCredentials
	}
	for _, permission := range p.requiredPermissions {
		if !permissions.RoleHasPermission(role, permission) {
			return authz.ErrNotAuthorized(fmt.Sprintf("not authorized to %s",
				proto.MarshalTextString(permission)))
		}
	}
	return nil
}

// With returns an authorizer that only authorizes users/tokens
// which satisfy all the given permissions.
func With(requiredPermissions ...*v1.Permission) authz.Authorizer {
	return &permissionChecker{requiredPermissions}
}
