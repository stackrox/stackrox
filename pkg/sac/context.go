package sac

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/utils"
)

type globalAccessScopeContextKey struct{}

// GlobalAccessScopeChecker retrieves the global access scope from the context.
// This function is guaranteed to return a non-nil value.
func GlobalAccessScopeChecker(ctx context.Context) ScopeChecker {
	core, _ := ctx.Value(globalAccessScopeContextKey{}).(ScopeCheckerCore)
	if core == nil {
		utils.Must(errors.New("global access scope was not found in context"))
		core = DenyAllAccessScopeChecker()
	}
	return NewScopeChecker(core)
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
