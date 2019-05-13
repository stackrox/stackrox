package permissioncheck

import (
	"context"

	"github.com/stackrox/rox/pkg/auth/permissions"
)

type contextKey struct{}

// ContextWithPermissionCheck returns a context that can be used to query an authorizer
// for the set of checked permissions.
func ContextWithPermissionCheck() (context.Context, permissions.PermissionMap) {
	perms := make(permissions.PermissionMap)
	return context.WithValue(context.Background(), contextKey{}, perms), perms
}

// FromContext retrieves a permission map (if any) used for performing
// a permission check from the given context.
func FromContext(ctx context.Context) permissions.PermissionMap {
	pc, _ := ctx.Value(contextKey{}).(permissions.PermissionMap)
	return pc
}
