package user

import (
	"context"
	"fmt"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
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
	id := authn.IdentityFromContext(ctx)
	if id == nil {
		return authz.ErrNoCredentials
	}

	// If sac scope checker is configured, skip role check.
	contextIsSACEnabled := sac.IsContextSACEnabled(ctx)
	contextIsSACV2Enabled := sac.IsContextSACV2Enabled(ctx)
	rootScopeChecker := sac.GlobalAccessScopeCheckerOrNil(ctx)

	// For plugin-based legacy SAC, only global permissions are checked here, and the plugin
	// is queried for these permissions.
	if contextIsSACEnabled && !contextIsSACV2Enabled && rootScopeChecker != nil {
		return p.checkGlobalSACPermissions(ctx, *rootScopeChecker)
	}
	// In all other cases (no SAC and builtin SAC v2), we check if the role has all the required permissions.
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
		return authz.ErrNotAuthorized("scoped access: not authorized")
	}
	return nil
}

func (p *permissionChecker) checkPermissions(perms *storage.ResourceToAccess) error {
	if perms == nil {
		return authz.ErrNoCredentials
	}
	for _, perm := range p.requiredPermissions {
		if !evaluateAgainstPermissions(perms, perm) {
			return authz.ErrNotAuthorized(fmt.Sprintf("not authorized to %s %s", perm.Access, perm.Resource))
		}
	}
	return nil
}

func evaluateAgainstPermissions(permissions *storage.ResourceToAccess, perm permissions.ResourceWithAccess) bool {
	return permissions.GetResourceToAccess()[string(perm.Resource.GetResource())] >= perm.Access
}
