package sac

import (
	"context"

	"github.com/stackrox/rox/pkg/sac/effectiveaccessscope"
)

// OneStepSCC is a ScopeCheckerCore implementation that looks at the first scope key
// and delegates to a different ScopeCheckerCore for the respective subscope.
type OneStepSCC map[ScopeKey]ScopeCheckerCore

// TryAllowed will return Deny for instances of this type.
func (c OneStepSCC) TryAllowed() TryAllowedResult {
	return Deny
}

// SubScopeChecker will return the ScopeCheckerCore for the given key, or a deny-all
// scope checker core if the key is not in the map.
func (c OneStepSCC) SubScopeChecker(key ScopeKey) ScopeCheckerCore {
	scc := c[key]
	if scc == nil {
		return denyAllScopeCheckerCore
	}
	return scc
}

// PerformChecks will call PerformChecks on all delegate scope checker cores.
func (c OneStepSCC) PerformChecks(ctx context.Context) error {
	for _, scc := range c {
		if err := scc.PerformChecks(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (c OneStepSCC) NeedsPostFiltering() bool {
	// The need for search result filtering can depend on the child-level need
	return true
}

func (c OneStepSCC) EffectiveAccessScope(_ context.Context) (*effectiveaccessscope.ScopeTree, error) {
	// Match the TryAllowed behaviour: Deny all
	return effectiveaccessscope.RestrictedEffectiveAccessScope(), nil
}
