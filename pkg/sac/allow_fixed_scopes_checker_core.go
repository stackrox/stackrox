package sac

import (
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/sac/effectiveaccessscope"
	"github.com/stackrox/rox/pkg/set"
)

type allowedFixedScopesCheckerCore struct {
	checkerLevel ScopeKind

	accessKeys    set.Set[AccessModeScopeKey]
	resourceKeys  set.Set[ResourceScopeKey]
	clusterKeys   set.Set[ClusterScopeKey]
	namespaceKeys set.Set[NamespaceScopeKey]

	accessLevel     AccessModeScopeKey
	targetResource  ResourceScopeKey
	targetCluster   ClusterScopeKey
	targetNamespace NamespaceScopeKey
}

func typedKeySet[T comparable](scopeKeys []ScopeKey) set.Set[T] {
	typedKeys := make([]T, 0, len(scopeKeys))
	for _, scopeKey := range scopeKeys {
		if key, ok := scopeKey.(T); ok {
			typedKeys = append(typedKeys, key)
		}
	}
	return set.NewSet[T](typedKeys...)
}

func allowFixedGlobalLevelScopes() ScopeCheckerCore {
	return &allowedFixedScopesCheckerCore{
		checkerLevel:  GlobalScopeKind,
		accessKeys:    typedKeySet[AccessModeScopeKey]([]ScopeKey{}),
		resourceKeys:  typedKeySet[ResourceScopeKey]([]ScopeKey{}),
		clusterKeys:   typedKeySet[ClusterScopeKey]([]ScopeKey{}),
		namespaceKeys: typedKeySet[NamespaceScopeKey]([]ScopeKey{}),
	}
}

func allowFixedAccessModeLevelScopes(
	accessLevelKeys []ScopeKey,
) ScopeCheckerCore {
	return &allowedFixedScopesCheckerCore{
		checkerLevel:  GlobalScopeKind,
		accessKeys:    typedKeySet[AccessModeScopeKey](accessLevelKeys),
		resourceKeys:  typedKeySet[ResourceScopeKey]([]ScopeKey{}),
		clusterKeys:   typedKeySet[ClusterScopeKey]([]ScopeKey{}),
		namespaceKeys: typedKeySet[NamespaceScopeKey]([]ScopeKey{}),
	}
}

func allowFixedResourceLevelScopes(
	accessLevelKeys []ScopeKey,
	resourceLevelKeys []ScopeKey,
) ScopeCheckerCore {
	return &allowedFixedScopesCheckerCore{
		checkerLevel:  GlobalScopeKind,
		accessKeys:    typedKeySet[AccessModeScopeKey](accessLevelKeys),
		resourceKeys:  typedKeySet[ResourceScopeKey](resourceLevelKeys),
		clusterKeys:   typedKeySet[ClusterScopeKey]([]ScopeKey{}),
		namespaceKeys: typedKeySet[NamespaceScopeKey]([]ScopeKey{}),
	}
}

func allowFixedClusterLevelScopes(
	accessLevelKeys []ScopeKey,
	resourceLevelKeys []ScopeKey,
	clusterLevelKeys []ScopeKey,
) ScopeCheckerCore {
	return &allowedFixedScopesCheckerCore{
		checkerLevel:  GlobalScopeKind,
		accessKeys:    typedKeySet[AccessModeScopeKey](accessLevelKeys),
		resourceKeys:  typedKeySet[ResourceScopeKey](resourceLevelKeys),
		clusterKeys:   typedKeySet[ClusterScopeKey](clusterLevelKeys),
		namespaceKeys: typedKeySet[NamespaceScopeKey]([]ScopeKey{}),
	}
}

func allowFixedNamespaceLevelScopes(
	accessLevelKeys []ScopeKey,
	resourceLevelKeys []ScopeKey,
	clusterLevelKeys []ScopeKey,
	namespaceLevelKeys []ScopeKey,
) ScopeCheckerCore {
	return &allowedFixedScopesCheckerCore{
		checkerLevel:  GlobalScopeKind,
		accessKeys:    typedKeySet[AccessModeScopeKey](accessLevelKeys),
		resourceKeys:  typedKeySet[ResourceScopeKey](resourceLevelKeys),
		clusterKeys:   typedKeySet[ClusterScopeKey](clusterLevelKeys),
		namespaceKeys: typedKeySet[NamespaceScopeKey](namespaceLevelKeys),
	}
}

func (c *allowedFixedScopesCheckerCore) SubScopeChecker(scopeKey ScopeKey) ScopeCheckerCore {
	switch key := scopeKey.(type) {
	case AccessModeScopeKey:
		if c.checkerLevel != GlobalScopeKind {
			return denyAllScopeCheckerCore
		}
		if c.accessKeys.Cardinality() > 0 && !c.accessKeys.Contains(key) {
			return denyAllScopeCheckerCore
		}
		return &allowedFixedScopesCheckerCore{
			checkerLevel:  AccessModeScopeKind,
			accessKeys:    c.accessKeys,
			resourceKeys:  c.resourceKeys,
			clusterKeys:   c.clusterKeys,
			namespaceKeys: c.namespaceKeys,
			accessLevel:   key,
		}
	case ResourceScopeKey:
		if c.checkerLevel != AccessModeScopeKind {
			return denyAllScopeCheckerCore
		}
		if c.resourceKeys.Cardinality() > 0 && !c.resourceKeys.Contains(key) {
			return denyAllScopeCheckerCore
		}
		return &allowedFixedScopesCheckerCore{
			checkerLevel:   ResourceScopeKind,
			accessKeys:     c.accessKeys,
			resourceKeys:   c.resourceKeys,
			clusterKeys:    c.clusterKeys,
			namespaceKeys:  c.namespaceKeys,
			accessLevel:    c.accessLevel,
			targetResource: key,
		}
	case ClusterScopeKey:
		if c.checkerLevel != ResourceScopeKind {
			return denyAllScopeCheckerCore
		}
		if c.clusterKeys.Cardinality() > 0 && !c.clusterKeys.Contains(key) {
			return denyAllScopeCheckerCore
		}
		return &allowedFixedScopesCheckerCore{
			checkerLevel:   ClusterScopeKind,
			accessKeys:     c.accessKeys,
			resourceKeys:   c.resourceKeys,
			clusterKeys:    c.clusterKeys,
			namespaceKeys:  c.namespaceKeys,
			accessLevel:    c.accessLevel,
			targetResource: c.targetResource,
			targetCluster:  key,
		}
	case NamespaceScopeKey:
		if c.checkerLevel != ClusterScopeKind {
			return denyAllScopeCheckerCore
		}
		if c.namespaceKeys.Cardinality() > 0 && !c.namespaceKeys.Contains(key) {
			return denyAllScopeCheckerCore
		}
		return &allowedFixedScopesCheckerCore{
			checkerLevel:    NamespaceScopeKind,
			accessKeys:      c.accessKeys,
			resourceKeys:    c.resourceKeys,
			clusterKeys:     c.clusterKeys,
			namespaceKeys:   c.namespaceKeys,
			accessLevel:     c.accessLevel,
			targetResource:  c.targetResource,
			targetCluster:   c.targetCluster,
			targetNamespace: key,
		}
	}
	return denyAllScopeCheckerCore
}

func (c *allowedFixedScopesCheckerCore) Allowed() bool {
	switch c.checkerLevel {
	case GlobalScopeKind:
		return c.accessKeys.Cardinality() == 0
	case AccessModeScopeKind:
		return c.resourceKeys.Cardinality() == 0
	case ResourceScopeKind:
		return c.clusterKeys.Cardinality() == 0
	case ClusterScopeKind:
		// resourceScope := resources.GetScopeForResource(permissions.Resource(c.targetResource))
		// if resourceScope == permissions.ClusterScope {
		// 	return true
		// }
		return c.namespaceKeys.Cardinality() == 0
	case NamespaceScopeKind:
		return true
	}
	return false
}

func (c *allowedFixedScopesCheckerCore) EffectiveAccessScope(
	resource permissions.ResourceWithAccess,
) (*effectiveaccessscope.ScopeTree, error) {
	if c.accessKeys.Cardinality() == 0 &&
		c.resourceKeys.Cardinality() == 0 &&
		c.clusterKeys.Cardinality() == 0 &&
		c.resourceKeys.Cardinality() == 0 {
		return effectiveaccessscope.UnrestrictedEffectiveAccessScope(), nil
	}
	if !c.accessKeys.Contains(AccessModeScopeKey(resource.Access)) {
		return effectiveaccessscope.DenyAllEffectiveAccessScope(), nil
	}
	if c.resourceKeys.Cardinality() == 0 &&
		c.clusterKeys.Cardinality() == 0 &&
		c.resourceKeys.Cardinality() == 0 {
		return effectiveaccessscope.UnrestrictedEffectiveAccessScope(), nil
	}
	targetResource := resource.Resource.GetResource()
	targetReplacingResource := resource.Resource.GetReplacingResource()
	if !c.resourceKeys.Contains(ResourceScopeKey(targetResource)) &&
		(targetReplacingResource == nil || !c.resourceKeys.Contains(ResourceScopeKey(*targetReplacingResource))) {
		return effectiveaccessscope.DenyAllEffectiveAccessScope(), nil
	}
	if c.clusterKeys.Cardinality() == 0 &&
		c.namespaceKeys.Cardinality() == 0 {
		return effectiveaccessscope.UnrestrictedEffectiveAccessScope(), nil
	}
	clusterIDs := make([]string, 0, c.clusterKeys.Cardinality())
	for _, clusterKey := range c.clusterKeys.AsSlice() {
		clusterIDs = append(clusterIDs, clusterKey.String())
	}
	if c.namespaceKeys.Cardinality() == 0 {
		return effectiveaccessscope.FromClustersAndNamespacesMap(clusterIDs, nil), nil
	}
	namespaces := make([]string, 0, c.namespaceKeys.Cardinality())
	for _, namespaceKey := range c.namespaceKeys.AsSlice() {
		namespaces = append(namespaces, namespaceKey.String())
	}
	namespaceMap := make(map[string][]string, len(clusterIDs))
	for clusterIx := range clusterIDs {
		clusterID := clusterIDs[clusterIx]
		namespaceMap[clusterID] = namespaces
	}
	return effectiveaccessscope.FromClustersAndNamespacesMap(nil, namespaceMap), nil
}

// AllowFixedScopes returns a scope checker core that allows those scopes that
// are in the cross product of all individual scope key lists. I.e.,
//
//	AllowFixedScopes(
//		AccessModeScopeKeys(storage.Access_READ, storage.Access_READ_WRITE),
//		ResourceScopeKeys(resources.CLUSTER),
//	)
//
// returns a scope checker core that allows read and write access to all cluster resources.
func AllowFixedScopes(keyLists ...[]ScopeKey) ScopeCheckerCore {
	switch len(keyLists) {
	case 0:
		return allowFixedGlobalLevelScopes()
	case 1:
		return allowFixedAccessModeLevelScopes(keyLists[0])
	case 2:
		return allowFixedResourceLevelScopes(keyLists[0], keyLists[1])
	case 3:
		return allowFixedClusterLevelScopes(keyLists[0], keyLists[1], keyLists[2])
	case 4:
		return allowFixedNamespaceLevelScopes(keyLists[0], keyLists[1], keyLists[2], keyLists[3])
	}
	return denyAllScopeCheckerCore
}
