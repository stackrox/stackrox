package sac

import (
	"context"

	"github.com/pkg/errors"
)

type globalAccessScopeContextKey struct{}
type sacEnabled struct{}
type sacV2Enabled struct{}

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

// IsContextSACV2Enabled will return true if SACv2 and SAC are enabled for a context and false otherwise
func IsContextSACV2Enabled(ctx context.Context) bool {
	contextHasSACv2 := ctx.Value(sacV2Enabled{})
	return contextHasSACv2 != nil && IsContextSACEnabled(ctx)
}

// SetContextSACV2Enabled enables SACv2 for a context.  Note, this must be done separately from setting a
// GlobalAccessScopeChecker for a context.  All contexts must have a GlobalAccessScopeChecker but not all contexts must
// have SAC enabled.
func SetContextSACV2Enabled(ctx context.Context) context.Context {
	return context.WithValue(SetContextSACEnabled(ctx), sacV2Enabled{}, struct{}{})
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
