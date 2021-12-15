package user

import (
	"context"
	"fmt"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/internal/permissioncheck"
	"github.com/stackrox/rox/pkg/sac"
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

	// If sac scope checker is configured, skip role check.
	contextIsSACEnabled := sac.IsContextSACEnabled(ctx)
	contextIsBuiltinScopedAuthzEnabled := sac.IsContextBuiltinScopedAuthzEnabled(ctx)
	rootScopeChecker := sac.GlobalAccessScopeCheckerOrNil(ctx)

	// For plugin-based legacy SAC, only global permissions are checked here,
	// and the plugin is queried for these permissions.
	if contextIsSACEnabled && !contextIsBuiltinScopedAuthzEnabled && rootScopeChecker != nil {
		return p.checkGlobalSACPermissions(ctx, *rootScopeChecker)
	}
	// In all other cases (no SAC and built-in scoped authorizer), we check if
	// the role has all the required permissions.
	return p.checkPermissions(id.Permissions())
}

func (p *permissionChecker) collectPermissions(pc *[]permissions.ResourceWithAccess) error {
	*pc = append(*pc, p.requiredPermissions...)
	return permissioncheck.ErrPermissionCheckOnly
}

func (p *permissionChecker) checkGlobalSACPermissions(ctx context.Context, rootSC sac.ScopeChecker) error {
	globalScopes := make([][]sac.ScopeKey, 0)
	for _, perm := range p.requiredPermissions {
		if !perm.Resource.PerformLegacyAuthForSAC() {
			continue
		}
		globalScopes = append(globalScopes, []sac.ScopeKey{
			sac.AccessModeScopeKey(perm.Access),
			sac.ResourceScopeKey(perm.Resource.GetResource()),
		})
	}

	allowed, err := rootSC.AllAllowed(ctx, globalScopes)
	if err != nil {
		return err
	}
	if !allowed {
		return errorhelpers.NewErrNotAuthorized("scoped access")
	}
	return nil
}

func (p *permissionChecker) checkPermissions(rolePerms map[string]storage.Access) error {
	if rolePerms == nil {
		return errorhelpers.ErrNoCredentials
	}
	for _, requiredPerm := range p.requiredPermissions {
		if !evaluateAgainstPermissions(rolePerms, requiredPerm) {
			return errorhelpers.NewErrNotAuthorized(fmt.Sprintf("%q for %q", requiredPerm.Access, requiredPerm.Resource))
		}
	}
	return nil
}

func evaluateAgainstPermissions(permissions map[string]storage.Access, perm permissions.ResourceWithAccess) bool {
	return permissions[string(perm.Resource.GetResource())] >= perm.Access
}
