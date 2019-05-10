package sac

import "context"

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
