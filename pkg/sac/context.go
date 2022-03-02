package sac

import (
	"context"

	"github.com/pkg/errors"
)

type globalAccessScopeContextKey struct{}
type pluginScopedAuthzEnabled struct{}

// GlobalAccessScopeChecker retrieves the global access scope from the context.
// This function is guaranteed to return a non-nil value.
func GlobalAccessScopeChecker(ctx context.Context) ScopeChecker {
	core, _ := ctx.Value(globalAccessScopeContextKey{}).(ScopeCheckerCore)
	if core == nil {
		core = ErrorAccessScopeCheckerCore(errors.New("global access scope was not found in context"))
	}
	return NewScopeChecker(core)
}

// IsContextPluginScopedAuthzEnabled will return true if the auth plugin scoped
// authorizer is enabled for a context and false otherwise.
func IsContextPluginScopedAuthzEnabled(ctx context.Context) bool {
	pluginScopedAuthz := ctx.Value(pluginScopedAuthzEnabled{})
	return pluginScopedAuthz != nil
}

// SetContextPluginScopedAuthzEnabled indicates the auth plugin scoped authorizer
// must be used in this context.
func SetContextPluginScopedAuthzEnabled(ctx context.Context) context.Context {
	return context.WithValue(ctx, pluginScopedAuthzEnabled{}, struct{}{})
}

// WithGlobalAccessScopeChecker returns a context that is a child of the given context and contains
// the given global access scope.
func WithGlobalAccessScopeChecker(ctx context.Context, as ScopeCheckerCore) context.Context {
	return context.WithValue(ctx, globalAccessScopeContextKey{}, as)
}

// WithAllAccess returns a context that is a child of the given context and will allow all access
// scope checks.
func WithAllAccess(ctx context.Context) context.Context {
	return WithGlobalAccessScopeChecker(ctx, allowAllScopeCheckerCore)
}

// WithNoAccess returns a context that is a child of the given context and will deny all access
// scope checks.
func WithNoAccess(ctx context.Context) context.Context {
	return WithGlobalAccessScopeChecker(ctx, denyAllScopeCheckerCore)
}
