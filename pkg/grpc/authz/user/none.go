package user

import (
	"context"

	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/grpc/authz"
)

// NoneRole identifies default role with no permissions
const NoneRole = "None"

// WithAnyRole returns an authorizer that only authorizes users/tokens with some role other than None
func WithAnyRole() authz.Authorizer {
	return &anyRoleChecker{}
}

type anyRoleChecker struct {
}

func (p *anyRoleChecker) Authorized(ctx context.Context, _ string) error {
	// Pull the identity from the context.
	id := authn.IdentityFromContext(ctx)
	if id == nil {
		return authz.ErrNoCredentials
	}
	for _, r := range id.Roles() {
		if r.GetRoleName() != NoneRole {
			return nil
		}
	}
	return authz.ErrNotAuthorized("The only role user has is None")
}
