package sac

import (
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/sac/effectiveaccessscope"
	"github.com/stackrox/rox/pkg/sac/resources"
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

// region ScopeCheckerCore interface functions

func (c *allowedFixedScopesCheckerCore) SubScopeChecker(scopeKey ScopeKey) ScopeCheckerCore {
	switch key := scopeKey.(type) {
	case AccessModeScopeKey:
		if c.checkerLevel != GlobalScopeKind {
			return denyAllScopeCheckerCore
		}
		if c.allowsGlobalAccess() {
			return c.subScopeCheckerBuilder().withAccessMode(key)
		}
		if !c.accessKeys.Contains(key) {
			return denyAllScopeCheckerCore
		}
		return c.subScopeCheckerBuilder().withAccessMode(key)
	case ResourceScopeKey:
		if c.checkerLevel != AccessModeScopeKind {
			return denyAllScopeCheckerCore
		}
		if c.allowsAccessModeLevelAccess() {
			return c.subScopeCheckerBuilder().withResource(key)
		}
		if !c.resourceKeys.Contains(key) {
			return denyAllScopeCheckerCore
		}
		return c.subScopeCheckerBuilder().withResource(key)
	case ClusterScopeKey:
		if c.checkerLevel != ResourceScopeKind {
			return denyAllScopeCheckerCore
		}
		if c.allowsResourceLevelAccess() {
			return c.subScopeCheckerBuilder().withCluster(key)
		}
		if !c.clusterKeys.Contains(key) {
			return denyAllScopeCheckerCore
		}
		return c.subScopeCheckerBuilder().withCluster(key)
	case NamespaceScopeKey:
		if c.checkerLevel != ClusterScopeKind {
			return denyAllScopeCheckerCore
		}
		if c.allowsClusterLevelAccess() {
			return c.subScopeCheckerBuilder().withNamespace(key)
		}
		if !c.namespaceKeys.Contains(key) {
			return denyAllScopeCheckerCore
		}
		return c.subScopeCheckerBuilder().withNamespace(key)
	default:
		return denyAllScopeCheckerCore
	}
}

func (c *allowedFixedScopesCheckerCore) Allowed() bool {
	resourceScope := resources.GetScopeForResource(permissions.Resource(c.targetResource))
	switch c.checkerLevel {
	case GlobalScopeKind:
		return c.allowsGlobalAccess()
	case AccessModeScopeKind:
		return c.allowsAccessModeLevelAccess()
	case ResourceScopeKind:
		if resourceScope == permissions.GlobalScope {
			return true
		}
		return c.allowsResourceLevelAccess()
	case ClusterScopeKind:
		if resourceScope == permissions.ClusterScope || resourceScope == permissions.GlobalScope {
			return true
		}
		return c.allowsClusterLevelAccess()
	case NamespaceScopeKind:
		return true
	default:
		return false
	}
}

func (c *allowedFixedScopesCheckerCore) EffectiveAccessScope(
	resource permissions.ResourceWithAccess,
) (*effectiveaccessscope.ScopeTree, error) {
	// Global access granted
	if c.allowsGlobalAccess() {
		return effectiveaccessscope.UnrestrictedEffectiveAccessScope(), nil
	}

	// Drill down to AccessMode level
	if !c.accessKeys.Contains(AccessModeScopeKey(resource.Access)) {
		return effectiveaccessscope.DenyAllEffectiveAccessScope(), nil
	}
	if c.allowsAccessModeLevelAccess() {
		return effectiveaccessscope.UnrestrictedEffectiveAccessScope(), nil
	}

	// Drill down to Resource level
	targetResource := resource.Resource.GetResource()
	targetReplacingResource := resource.Resource.GetReplacingResource()
	if !c.resourceKeys.Contains(ResourceScopeKey(targetResource)) &&
		(targetReplacingResource == nil || !c.resourceKeys.Contains(ResourceScopeKey(*targetReplacingResource))) {
		return effectiveaccessscope.DenyAllEffectiveAccessScope(), nil
	}
	if c.allowsResourceLevelAccess() {
		return effectiveaccessscope.UnrestrictedEffectiveAccessScope(), nil
	}

	// Cluster and Namespace level
	clusterIDs := make([]string, 0, c.clusterKeys.Cardinality())
	for clusterKey := range c.clusterKeys {
		clusterIDs = append(clusterIDs, clusterKey.String())
	}
	if c.allowsClusterLevelAccess() {
		return effectiveaccessscope.FromClustersAndNamespacesMap(clusterIDs, nil), nil
	}
	namespaces := make([]string, 0, c.namespaceKeys.Cardinality())
	for namespaceKey := range c.namespaceKeys {
		namespaces = append(namespaces, namespaceKey.String())
	}
	clusterNamespaceMap := make(map[string][]string, len(clusterIDs))
	for _, clusterID := range clusterIDs {
		clusterNamespaceMap[clusterID] = namespaces
	}
	return effectiveaccessscope.FromClustersAndNamespacesMap(nil, clusterNamespaceMap), nil
}

// endregion ScopeCheckerCore interface functions

// region Public constructors

// AllowFixedGlobalLevelScopes returns a scope checker core that allows those scopes that
// are in the cross product of all access and resource individual scope key lists. I.e.,
//
//	AllowFixedGlobalLevelScopes()
//
// returns a scope checker core that allows read and write access to all cluster resources.
func AllowFixedGlobalLevelScopes() ScopeCheckerCore {
	return &allowedFixedScopesCheckerCore{
		checkerLevel:  GlobalScopeKind,
		accessKeys:    set.NewSet[AccessModeScopeKey](),
		resourceKeys:  set.NewSet[ResourceScopeKey](),
		clusterKeys:   set.NewSet[ClusterScopeKey](),
		namespaceKeys: set.NewSet[NamespaceScopeKey](),
	}
}

// AllowFixedAccessLevelScopes returns a scope checker core that allows those scopes that
// are in the cross product of all access and resource individual scope key lists. I.e.,
//
//	 AllowFixedAccessLevelScopes(
//			AccessModeScopeKeys(storage.Access_READ, storage.Access_READ_WRITE),
//	 )
//
// returns a scope checker core that allows read and write access to all cluster resources.
func AllowFixedAccessLevelScopes(
	accessLevelKeys []AccessModeScopeKey,
) ScopeCheckerCore {
	return &allowedFixedScopesCheckerCore{
		checkerLevel:  GlobalScopeKind,
		accessKeys:    set.NewSet(accessLevelKeys...),
		resourceKeys:  set.NewSet[ResourceScopeKey](),
		clusterKeys:   set.NewSet[ClusterScopeKey](),
		namespaceKeys: set.NewSet[NamespaceScopeKey](),
	}
}

// AllowFixedResourceLevelScopes returns a scope checker core that allows those scopes that
// are in the cross product of all access and resource individual scope key lists. I.e.,
//
//	 AllowFixedResourceLevelScopes(
//			AccessModeScopeKeys(storage.Access_READ, storage.Access_READ_WRITE),
//			ResourceScopeKeys(resources.CLUSTER),
//	 )
//
// returns a scope checker core that allows read and write access to all cluster resources.
func AllowFixedResourceLevelScopes(
	accessLevelKeys []AccessModeScopeKey,
	resourceLevelKeys []ResourceScopeKey,
) ScopeCheckerCore {
	return &allowedFixedScopesCheckerCore{
		checkerLevel:  GlobalScopeKind,
		accessKeys:    set.NewSet(accessLevelKeys...),
		resourceKeys:  set.NewSet(resourceLevelKeys...),
		clusterKeys:   set.NewSet[ClusterScopeKey](),
		namespaceKeys: set.NewSet[NamespaceScopeKey](),
	}
}

// AllowFixedClusterLevelScopes returns a scope checker core that allows those scopes that
// are in the cross product of all access and resource individual scope key lists. I.e.,
//
//		 AllowFixedClusterLevelScopes(
//				AccessModeScopeKeys(storage.Access_READ, storage.Access_READ_WRITE),
//				ResourceScopeKeys(resources.CLUSTER),
//	         ClusterScopeKeys(clusterID1, clusterID2),
//		 )
//
// returns a scope checker core that allows read and write access to all cluster resources
// within cluster1 and cluster2.
func AllowFixedClusterLevelScopes(
	accessLevelKeys []AccessModeScopeKey,
	resourceLevelKeys []ResourceScopeKey,
	clusterLevelKeys []ClusterScopeKey,
) ScopeCheckerCore {
	return &allowedFixedScopesCheckerCore{
		checkerLevel:  GlobalScopeKind,
		accessKeys:    set.NewSet(accessLevelKeys...),
		resourceKeys:  set.NewSet(resourceLevelKeys...),
		clusterKeys:   set.NewSet(clusterLevelKeys...),
		namespaceKeys: set.NewSet[NamespaceScopeKey](),
	}
}

// AllowFixedNamespaceLevelScopes returns a scope checker core that allows those scopes that
// are in the cross product of all access and resource individual scope key lists. I.e.,
//
//		 AllowFixedNamespaceLevelScopes(
//				AccessModeScopeKeys(storage.Access_READ, storage.Access_READ_WRITE),
//				ResourceScopeKeys(resources.CLUSTER),
//	         ClusterScopeKeys(clusterID1, clusterID2),
//	         NamespaceScopeKeys(namespace1, namespace2),
//		 )
//
// returns a scope checker core that allows read and write access to all cluster resources within
// cluster1 namespace1, cluster1 namespace2, cluster2 namespace1 and cluster2 namespace2.
func AllowFixedNamespaceLevelScopes(
	accessLevelKeys []AccessModeScopeKey,
	resourceLevelKeys []ResourceScopeKey,
	clusterLevelKeys []ClusterScopeKey,
	namespaceLevelKeys []NamespaceScopeKey,
) ScopeCheckerCore {
	return &allowedFixedScopesCheckerCore{
		checkerLevel:  GlobalScopeKind,
		accessKeys:    set.NewSet(accessLevelKeys...),
		resourceKeys:  set.NewSet(resourceLevelKeys...),
		clusterKeys:   set.NewSet(clusterLevelKeys...),
		namespaceKeys: set.NewSet(namespaceLevelKeys...),
	}
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
	default:
		return denyAllScopeCheckerCore
	}
}

// endregion Public constructors

// region helpers for Interface functions

type subScopeCheckerBuilder struct {
	core *allowedFixedScopesCheckerCore
}

func (c *allowedFixedScopesCheckerCore) subScopeCheckerBuilder() *subScopeCheckerBuilder {
	return &subScopeCheckerBuilder{
		core: &allowedFixedScopesCheckerCore{
			checkerLevel:    c.checkerLevel + 1,
			accessKeys:      c.accessKeys,
			resourceKeys:    c.resourceKeys,
			clusterKeys:     c.clusterKeys,
			namespaceKeys:   c.namespaceKeys,
			accessLevel:     c.accessLevel,
			targetResource:  c.targetResource,
			targetCluster:   c.targetCluster,
			targetNamespace: c.targetNamespace,
		},
	}
}

func (b *subScopeCheckerBuilder) withAccessMode(key AccessModeScopeKey) ScopeCheckerCore {
	b.core.accessLevel = key
	return b.core
}

func (b *subScopeCheckerBuilder) withResource(key ResourceScopeKey) ScopeCheckerCore {
	b.core.targetResource = key
	return b.core
}

func (b *subScopeCheckerBuilder) withCluster(key ClusterScopeKey) ScopeCheckerCore {
	b.core.targetCluster = key
	return b.core
}

func (b *subScopeCheckerBuilder) withNamespace(key NamespaceScopeKey) ScopeCheckerCore {
	b.core.targetNamespace = key
	return b.core
}

func (c *allowedFixedScopesCheckerCore) allowsGlobalAccess() bool {
	return c.accessKeys.Cardinality() == 0 && c.allowsAccessModeLevelAccess()
}

func (c *allowedFixedScopesCheckerCore) allowsAccessModeLevelAccess() bool {
	return c.resourceKeys.Cardinality() == 0 && c.allowsResourceLevelAccess()
}

func (c *allowedFixedScopesCheckerCore) allowsResourceLevelAccess() bool {
	return c.clusterKeys.Cardinality() == 0 && c.allowsClusterLevelAccess()
}

func (c *allowedFixedScopesCheckerCore) allowsClusterLevelAccess() bool {
	return c.namespaceKeys.Cardinality() == 0
}

// endregion helpers for Interface functions

// region helpers for constructors

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

// endregion helpers for constructors
