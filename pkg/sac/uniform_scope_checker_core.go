package sac

import (
	"context"
)

type uniformScopeCheckerCore bool

var (
	denyAllScopeCheckerCore  = uniformScopeCheckerCore(false)
	allowAllScopeCheckerCore = uniformScopeCheckerCore(true)
)

// DenyAllAccessScopeChecker returns an access scope checker that denies access to all scopes.
func DenyAllAccessScopeChecker() ScopeCheckerCore {
	return denyAllScopeCheckerCore
}

// AllowAllAccessScopeChecker returns an access scope checker that allows access to all scopes.
func AllowAllAccessScopeChecker() ScopeCheckerCore {
	return allowAllScopeCheckerCore
}

func (s uniformScopeCheckerCore) SubScopeChecker(key ScopeKey) ScopeCheckerCore {
	return s
}

func (s uniformScopeCheckerCore) TryAllowed() TryAllowedResult {
	if s {
		return Allow
	}
	return Deny
}

func (s uniformScopeCheckerCore) PerformChecks(ctx context.Context) error {
	return nil
}
