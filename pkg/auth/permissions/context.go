package permissions

import (
	"context"
)

type resolvedRolesContextKey struct{}

func ContextWithResolvedRoles(ctx context.Context, resolvedRoles []ResolvedRole) context.Context {
	return context.WithValue(ctx, resolvedRolesContextKey{}, resolvedRoles)
}

func ResolvedRolesFromContext(ctx context.Context) []ResolvedRole {
	return ctx.Value(resolvedRolesContextKey{}).([]ResolvedRole)
}
