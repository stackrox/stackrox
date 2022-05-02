package user

import (
	"context"

	"github.com/stackrox/rox/pkg/auth"
	"github.com/stackrox/rox/pkg/auth/permissions/utils"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/grpc/authz"
)

// WithRole returns an authorizer that only authorizes users/tokens with Admin role
func WithRole(roleName string) authz.Authorizer {
	return &roleChecker{roleName}
}

type roleChecker struct {
	roleName string
}

func (p *roleChecker) Authorized(ctx context.Context, _ string) error {
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

	for _, roleName := range roleNames {
		if roleName == p.roleName {
			return nil
		}
	}

	return errox.NotAuthorized.CausedByf("role %q is required", p.roleName)
}
