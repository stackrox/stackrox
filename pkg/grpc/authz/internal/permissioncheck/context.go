package permissioncheck

import (
	"context"

	"github.com/stackrox/rox/pkg/auth/permissions"
)

type contextKey struct{}

// FromContext retrieves a permission map (if any) used for performing
// a permission check from the given context.
func FromContext(ctx context.Context) *[]permissions.ResourceWithAccess {
	pc, _ := ctx.Value(contextKey{}).(*[]permissions.ResourceWithAccess)
	return pc
}
