package scoped

import (
	"context"

	v1 "github.com/stackrox/stackrox/generated/api/v1"
)

// Scope hold an id and scope level for scoping searches.
type Scope struct {
	ID     string
	Level  v1.SearchCategory
	Parent *Scope
}

// scopedContextKey is the key for the scope value in the context.
type scopedContextKey struct{}

// scopedContextValue holds the value of the scope in the context.
type scopedContextValue struct {
	scope Scope
}

// Context returns a new context with the scope attached.
func Context(ctx context.Context, scope Scope) context.Context {
	inner, ok := ctx.Value(scopedContextKey{}).(*scopedContextValue)
	if ok {
		scope.Parent = &inner.scope
	}
	return context.WithValue(ctx, scopedContextKey{}, &scopedContextValue{
		scope: scope,
	})
}

// GetScope returns the Scope from the input context as well as a boolean indicating if there was a Scope attached.
func GetScope(hasGraphContext context.Context) (Scope, bool) {
	if hasGraphContext == nil {
		return Scope{}, false
	}
	inter := hasGraphContext.Value(scopedContextKey{})
	if inter == nil {
		return Scope{}, false
	}
	s := inter.(*scopedContextValue)
	return s.scope, true
}

// GetScopeAtLevel returns the Scope from the input context with the given level, nil if that level doesn't exist in the scope hierarchy
func GetScopeAtLevel(hasGraphContext context.Context, level v1.SearchCategory) (Scope, bool) {
	if hasGraphContext == nil {
		return Scope{}, false
	}
	inter := hasGraphContext.Value(scopedContextKey{})
	if inter == nil {
		return Scope{}, false
	}
	scope := inter.(*scopedContextValue).scope
	return getScopeAtLevel(&scope, level)
}

func getScopeAtLevel(scope *Scope, level v1.SearchCategory) (Scope, bool) {
	if scope == nil {
		return Scope{}, false
	}
	if scope.Level == level {
		return *scope, true
	}
	return getScopeAtLevel(scope.Parent, level)
}
