package user

import (
	"context"
	"slices"

	"github.com/stackrox/rox/pkg/auth"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/auth/permissions/utils"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/grpc/authz"
)

// WithRole returns an authorizer that only authorizes users/tokens with Admin role
func WithRole(roleName string) authz.Authorizer {
	return &roleChecker{roleName}
}

var _ authz.Authorizer = (*roleChecker)(nil)

type roleChecker struct {
	roleName string
}

// EnrichContext returns the input context unchanged.
func (p *roleChecker) EnrichContext(ctx context.Context, _ string) context.Context {
	if features.ACMAccessControlDelegation.Enabled() {
		id, err := authn.IdentityFromContext(ctx)
		if err != nil {
			return ctx
		}
		authProvider := id.ExternalAuthProvider()
		roleMapper := authProvider.RoleMapper()
		resolvedRoles, err := roleMapper.FromUserDescriptor(ctx, id.UserDescriptor())
		if err != nil {
			return ctx
		}
		// The following block helps with restricting the actual user roles to the minimal set
		// needed for the service request.
		filteredRoles := make([]permissions.ResolvedRole, 0, 1)
		for _, role := range resolvedRoles {
			if role.GetRoleName() == p.roleName {
				filteredRoles = append(filteredRoles, role)
			}
		}
		return permissions.ContextWithResolvedRoles(ctx, filteredRoles)
	}
	return ctx
}

func (p *roleChecker) Authorized(ctx context.Context, _ string) error {
	if features.ACMAccessControlDelegation.Enabled() {
		userRoles := permissions.ResolvedRolesFromContext(ctx)
		return p.checkRole(utils.RoleNames(userRoles))
	}
	id, err := authn.IdentityFromContext(ctx)
	if err != nil {
		return err
	}

	return p.checkRole(utils.RoleNames(id.Roles()))
}

func (p *roleChecker) checkRole(roleNames []string) error {
	if len(roleNames) == 0 {
		return auth.ErrNoValidRole
	}

	if slices.Contains(roleNames, p.roleName) {
		return nil
	}

	return errox.NotAuthorized.CausedByf("role %q is required", p.roleName)
}
