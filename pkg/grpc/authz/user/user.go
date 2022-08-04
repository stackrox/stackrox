package user

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/internal/permissioncheck"
)

// With returns an authorizer that only authorizes users/tokens
// which satisfy all the given permissions.
func With(requiredPermissions ...permissions.ResourceWithAccess) authz.Authorizer {
	return &permissionChecker{requiredPermissions}
}

type permissionChecker struct {
	requiredPermissions []permissions.ResourceWithAccess
}

func (p *permissionChecker) Authorized(ctx context.Context, _ string) error {
	// If we are just fetching the permissions needed.
	if pc := permissioncheck.FromContext(ctx); pc != nil {
		return p.collectPermissions(pc)
	}

	// Pull the identity from the context.
	id, err := authn.IdentityFromContext(ctx)
	if err != nil {
		return err
	}

	// Check if the role has all the required permissions.
	return p.checkPermissions(id.Permissions())
}

func (p *permissionChecker) collectPermissions(pc *[]permissions.ResourceWithAccess) error {
	*pc = append(*pc, p.requiredPermissions...)
	return permissioncheck.ErrPermissionCheckOnly
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
