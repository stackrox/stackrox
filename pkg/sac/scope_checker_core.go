package sac

import (
	"context"

	"github.com/stackrox/default-authz-plugin/pkg/payload"
	"github.com/stackrox/stackrox/pkg/auth/permissions"
	"github.com/stackrox/stackrox/pkg/sac/effectiveaccessscope"
)

// TryAllowedResult represents the possible values of a `TryAllowed` call on an access scope checker.
//go:generate stringer -type=TryAllowedResult
type TryAllowedResult int32

const (
	// Unknown indicates that the value is not known and must be resolved via a call to `PerformChecks`.
	Unknown TryAllowedResult = iota
	// Deny indicates that access to the given scope is not allowed.
	Deny
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
//go:generate mockgen-wrapper
type ScopeCheckerCore interface {
	// SubScopeChecker obtains an access scope checker for the access scope directly underneath
	// this scope, keyed by the given key.
	SubScopeChecker(scopeKey ScopeKey) ScopeCheckerCore
	// TryAllowed checks if access to the scope being checked is allowed. If the result is `Unknown`, the
	// attempt is recorded and the corresponding data will be queried in the next call to `PerformChecks`.
	TryAllowed() TryAllowedResult
	// PerformChecks queries the Authorization Plugin for the set of access scopes underneath (and including)
	// this scope for which `TryAllowed` calls have returned `Unknown` previously.
	// Note: Only scopes that have been obtained from this scope via a call to `SubScopeChecker` are guaranteed
	// to be considered. Similarly, only requests made in the current goroutine are guaranteed to be considered.
	PerformChecks(ctx context.Context) error
	// EffectiveAccessScope returns effective access scope for given principal stored in context.
	// If checker is not at resource level then it returns an error.
	EffectiveAccessScope(resource permissions.ResourceWithAccess) (*effectiveaccessscope.ScopeTree, error)
}

// NewRootScopeCheckerCore returns a ScopeCheckerCore with a root AccessScope
func NewRootScopeCheckerCore(reqTracker ScopeRequestTracker) ScopeCheckerCore {
	return NewScopeCheckerCore(payload.AccessScope{}, reqTracker)
}

// NewScopeCheckerCore returns a new ScopeCheckerCore for a given scope
func NewScopeCheckerCore(currentScope payload.AccessScope, reqTracker ScopeRequestTracker) ScopeCheckerCore {
	scc := &ScopeCheckerCoreImpl{
		children:     make(map[ScopeKey]ScopeCheckerCore),
		currentScope: currentScope,
		reqTracker:   reqTracker,
		state:        int32(Unknown),
	}
	return scc
}
