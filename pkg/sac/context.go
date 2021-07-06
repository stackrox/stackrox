package sac

import (
	"context"

	"github.com/pkg/errors"
)

type globalAccessScopeContextKey struct{}
type sacEnabled struct{}
type builtinScopedAuthzEnabled struct{}

// GlobalAccessScopeChecker retrieves the global access scope from the context.
// This function is guaranteed to return a non-nil value.
func GlobalAccessScopeChecker(ctx context.Context) ScopeChecker {
	core, _ := ctx.Value(globalAccessScopeContextKey{}).(ScopeCheckerCore)
	if core == nil {
		core = ErrorAccessScopeCheckerCore(errors.New("global access scope was not found in context"))
	}
	return NewScopeChecker(core)
}

// GlobalAccessScopeCheckerOrNil retrieves the global access scope checker if it is stored in the context; otherwise,
// it returns `nil`.
func GlobalAccessScopeCheckerOrNil(ctx context.Context) *ScopeChecker {
	core, _ := ctx.Value(globalAccessScopeContextKey{}).(ScopeCheckerCore)
	if core == nil {
		return nil
	}
	sc := NewScopeChecker(core)
	return &sc
}

// IsContextSACEnabled will return true if SAC is enabled for a context and false if SAC has not been enabled for a
// context
func IsContextSACEnabled(ctx context.Context) bool {
	if contextHasSAC := ctx.Value(sacEnabled{}); contextHasSAC != nil {
		return true
	}
	return false
}

// SetContextSACEnabled enables SAC for a context.  Note, this must be done separately from setting a
// GlobalAccessScopeChecker for a context.  All contexts must have a GlobalAccessScopeChecker but not all contexts must
// have SAC enabled.
func SetContextSACEnabled(ctx context.Context) context.Context {
	return context.WithValue(ctx, sacEnabled{}, struct{}{})
}

// IsContextBuiltinScopedAuthzEnabled will return true if the built-in scoped
// authorizer and SAC are enabled for a context and false otherwise.
func IsContextBuiltinScopedAuthzEnabled(ctx context.Context) bool {
	builtinScopedAuthz := ctx.Value(builtinScopedAuthzEnabled{})
	return builtinScopedAuthz != nil && IsContextSACEnabled(ctx)
}

// SetContextBuiltinScopedAuthzEnabled indicates the built-in scoped authorizer
// must be used in this context. This must be done separately from setting a
// GlobalAccessScopeChecker for a context. All contexts must have a
// GlobalAccessScopeChecker but not all contexts must use the built-in scoped
// authorizer.
func SetContextBuiltinScopedAuthzEnabled(ctx context.Context) context.Context {
	return context.WithValue(SetContextSACEnabled(ctx), builtinScopedAuthzEnabled{}, struct{}{})
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
