package user

import (
	"context"
	"fmt"

	"github.com/gogo/protobuf/proto"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/internal/permissioncheck"
)

type permissionChecker struct {
	requiredPermissions []*v1.Permission
}

func (p *permissionChecker) Authorized(ctx context.Context, _ string) error {
	if pc := permissioncheck.FromContext(ctx); pc != nil {
		for _, perm := range p.requiredPermissions {
			pc.Add(permissions.Resource(perm.Resource), perm.Access)
		}

		return permissioncheck.ErrPermissionCheckOnly
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
