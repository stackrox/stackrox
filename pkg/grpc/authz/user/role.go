package user

import (
	"context"
	"fmt"

	"github.com/stackrox/rox/generated/storage"
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
	// Pull the identity from the context.
	id := authn.IdentityFromContext(ctx)
	if id == nil {
		return authz.ErrNoCredentials
	}

	return p.checkRole(id.Roles())
}

func (p *roleChecker) checkRole(roles []*storage.Role) error {
	if len(roles) == 0 {
		return authz.ErrNoCredentials
	}

	for _, role := range roles {
		if role.GetName() == p.roleName {
			return nil
		}
	}

	return authz.ErrNotAuthorized(fmt.Sprintf("role %q is required", p.roleName))
}
