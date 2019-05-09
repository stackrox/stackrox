package sac

import "context"

type scopeKeySet map[ScopeKey]struct{}

type allowFixedScopesCheckerCore []scopeKeySet

// AllowFixedScopes returns a scope checker core that allows those scopes that
// are in the cross product of all individual scope key lists. I.e.,
// AllowFixedScopes(
//   AccessModeScopeKeys(storage.Access_READ, storage.Access_READ_WRITE),
//   ResourceScopeKeys(resources.CLUSTER),
// )
// returns a scope checker core that allows read and write access to all cluster resources.
func AllowFixedScopes(keyLists ...[]ScopeKey) ScopeCheckerCore {
	sets := make(allowFixedScopesCheckerCore, len(keyLists))
	for i, keyList := range keyLists {
		set := make(map[ScopeKey]struct{}, len(keyList))
		for _, key := range keyList {
			set[key] = struct{}{}
		}
		sets[i] = set
	}
	return sets
}

func (c allowFixedScopesCheckerCore) TryAllowed() TryAllowedResult {
	if len(c) == 0 {
		return Allow
	}
	return Deny
}

func (c allowFixedScopesCheckerCore) PerformChecks(_ context.Context) error {
	return nil
}

func (c allowFixedScopesCheckerCore) SubScopeChecker(key ScopeKey) ScopeCheckerCore {
	if len(c) == 0 {
		return c
	}
	if _, ok := c[0][key]; ok {
		return c[1:]
	}
	return denyAllScopeCheckerCore
}
