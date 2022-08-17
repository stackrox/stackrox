package sac

import (
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/sac/effectiveaccessscope"
)

// TryAllowedResult represents the possible values of a `TryAllowed` call on an access scope checker.
//
//go:generate stringer -type=TryAllowedResult
type TryAllowedResult int32

const (
	// Deny indicates that access to the given scope is not allowed.
	Deny TryAllowedResult = iota
	// Allow indicates that access to the given scope is allowed.
	Allow
)

// ScopeCheckerCore represents an interface for querying access to given scopes.
// As the name `Core` indicates, this interface is designed to be implemented for custom
// access scope check logic, but not to be used directly by clients. For this, use the
// `ScopeChecker` type.
// Note: This interface does not provide any information about the scope it represents. This is
// intentional to allow efficient implementations for special access scope checkers such as
// "allow all".
//
//go:generate mockgen-wrapper
type ScopeCheckerCore interface {
	// SubScopeChecker obtains an access scope checker for the access scope directly underneath
	// this scope, keyed by the given key.
	SubScopeChecker(scopeKey ScopeKey) ScopeCheckerCore
	// TryAllowed checks if access to the scope being checked is allowed.
	TryAllowed() TryAllowedResult
	// EffectiveAccessScope returns effective access scope for given principal stored in context.
	// If checker is not at resource level then it returns an error.
	EffectiveAccessScope(resource permissions.ResourceWithAccess) (*effectiveaccessscope.ScopeTree, error)
}
