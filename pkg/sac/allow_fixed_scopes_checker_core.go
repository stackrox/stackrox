package sac

import (
	"context"

	"github.com/stackrox/rox/pkg/sac/effectiveaccessscope"
)

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

func (c allowFixedScopesCheckerCore) NeedsPostFiltering() bool {
	return true
	// TODO: toggle the logic to that one once the scope filter extraction is implemented
	// The complete allowed scope is known and can be injected as request
	// filter. Post-filtering is not needed
	//return false
}

func (c allowFixedScopesCheckerCore) EffectiveAccessScope(_ context.Context) (*effectiveaccessscope.ScopeTree, error) {
	// EffectiveAccessScope should only be called on a core of resource level
	// Here we only know the level from the type of keys describing the children levels.
	// The next level is described by c[0]
	// 1. If no child core, match the TryAllowed Allow strategy
	// 2. Cluster level sub-cores, build a partial ScopeTree and add each cluster subtree
	// 3. Other level sub-cores,deny all
	if len(c) == 0 {
		return effectiveaccessscope.UnrestrictedEffectiveAccessScope(), nil
	}
	hasClusterLevelChildren := false
	for k, _ := range c[0] {
		switch k.ScopeKind() {
		case ClusterScopeKind:
			hasClusterLevelChildren = true
		default:
			hasClusterLevelChildren = false
		}
		break
	}
	if !hasClusterLevelChildren {
		return effectiveaccessscope.RestrictedEffectiveAccessScope(), nil
	}
	clusterIDs := make([]string, 0, len(c[0]))
	namespaces := make([]string, 0)
	for k, _ := range c[0] {
		clusterIDs = append(clusterIDs, k.String())
	}
	if len(c) > 1 {
		for k, _ := range c[1] {
			namespaces = append(namespaces, k.String())
		}
	}
	return effectiveaccessscope.FromClusterIDsAndNamespaces(clusterIDs, namespaces), nil
}
