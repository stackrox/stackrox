package user

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/grpc/authz"
)

// With returns an authorizer that only authorizes users/tokens
// which satisfy all the given permissions.
func With(requiredPermissions ...permissions.ResourceWithAccess) authz.Authorizer {
	return &permissionChecker{requiredPermissions}
}

// Authenticated returns an authorizer that only authorizers authenticated
// users/tokens regardless what permissions they have.
func Authenticated() authz.Authorizer {
	// Relies on With implementation detail: no required permissions,
	// no checks to perform.
	return With()
}

type permissionChecker struct {
	requiredPermissions []permissions.ResourceWithAccess
}

func (p *permissionChecker) Authorized(ctx context.Context, _ string) error {
	// Pull the identity from the context.
	id, err := authn.IdentityFromContext(ctx)
	if err != nil {
		return err
	}

	// Check if the role has all the required permissions.
	return p.checkPermissions(id.Permissions())
}

func (p *permissionChecker) checkPermissions(rolePerms map[string]storage.Access) error {
	if rolePerms == nil {
		return errox.NoCredentials
	}
	for _, requiredPerm := range p.requiredPermissions {
		if !evaluateAgainstPermissions(rolePerms, requiredPerm) {
			return errox.NotAuthorized.CausedByf("%q for %q", requiredPerm.Access, requiredPerm.Resource)
		}
	}
	return nil
}

func evaluateAgainstPermissions(perms map[string]storage.Access, perm permissions.ResourceWithAccess) bool {
	return perm.Resource.IsPermittedBy(perms, perm.Access)
}
