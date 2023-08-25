package sac

import (
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/sac/effectiveaccessscope"
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

func (s uniformScopeCheckerCore) SubScopeChecker(_ ScopeKey) ScopeCheckerCore {
	return s
}

func (s uniformScopeCheckerCore) Allowed() bool {
	return bool(s)
}

func (s uniformScopeCheckerCore) EffectiveAccessScope(_ permissions.ResourceWithAccess) (*effectiveaccessscope.ScopeTree, error) {
	if s {
		return effectiveaccessscope.UnrestrictedEffectiveAccessScope(), nil
	}
	return effectiveaccessscope.DenyAllEffectiveAccessScope(), nil
}
