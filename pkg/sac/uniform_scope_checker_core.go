package sac

import (
	"context"

	"github.com/stackrox/stackrox/pkg/auth/permissions"
	"github.com/stackrox/stackrox/pkg/sac/effectiveaccessscope"
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

func (s uniformScopeCheckerCore) EffectiveAccessScope(resource permissions.ResourceWithAccess) (*effectiveaccessscope.ScopeTree, error) {
	if s {
		return effectiveaccessscope.UnrestrictedEffectiveAccessScope(), nil
	}
	return effectiveaccessscope.DenyAllEffectiveAccessScope(), nil
}
