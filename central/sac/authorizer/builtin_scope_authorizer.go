package authorizer

import (
	"context"

	"github.com/pkg/errors"
	clusterStore "github.com/stackrox/rox/central/cluster/datastore"
	namespaceStore "github.com/stackrox/rox/central/namespace/datastore"
	"github.com/stackrox/rox/central/role/resources"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	// ErrUnexpectedScopeKey is returned when scope key does not match expected level.
	ErrUnexpectedScopeKey = errors.New("unexpected scope key")
	// ErrUnknownResource is returned when resource is unknown.
	ErrUnknownResource = errors.New("unknown resource")
)

// NewBuiltInScopeChecker returns a new SAC scope checker for a given scope
func NewBuiltInScopeChecker(ctx context.Context, roles []permissions.ResolvedRole) (sac.ScopeCheckerCore, error) {
	adminCtx := sac.WithGlobalAccessScopeChecker(ctx, sac.AllowAllAccessScopeChecker())

	clusters, err := clusterStore.Singleton().GetClusters(adminCtx)
	if err != nil {
		return nil, errors.Wrap(err, "reading all clusters")
	}
	namespaces, err := namespaceStore.Singleton().GetNamespaces(adminCtx)
	if err != nil {
		return nil, errors.Wrap(err, "reading all namespaces")
	}

	return newGlobalScopeCheckerCore(clusters, namespaces, roles), nil
}

func newGlobalScopeCheckerCore(clusters []*storage.Cluster, namespaces []*storage.NamespaceMetadata, roles []permissions.ResolvedRole) sac.ScopeCheckerCore {
	scc := &globalScopeChecker{
		roles: roles,
		cache: &authorizerDataCache{
			clusters:                  clusters,
			namespaces:                namespaces,
			effectiveAccessScopesByID: make(map[string]*sac.EffectiveAccessScopeTree),
		},
	}
	return scc
}

// globalScopeCheckerCore maintains a list of resolved roles only.
// TryAllowed always returns Deny since narrower scope is required to decide if request should be allowed.
// This simplifies logic as only user with admin rights can be allowed on global scope.
// PerformChecks does nothing since there is nothing to check.
// SubScopeChecker extracts the access mode from the scope key,
// and returns an accessModeLevelScopeCheckerCore with the same resolved roles,
//setting the desired access mode.
type globalScopeChecker struct {
	cache *authorizerDataCache

	roles []permissions.ResolvedRole
}

func (a *globalScopeChecker) TryAllowed() sac.TryAllowedResult {
	return sac.Deny
}

func (a *globalScopeChecker) PerformChecks(_ context.Context) error {
	return nil
}

func (a *globalScopeChecker) SubScopeChecker(scopeKey sac.ScopeKey) sac.ScopeCheckerCore {
	scope, ok := scopeKey.(sac.AccessModeScopeKey)
	if !ok {
		return errorScopeChecker(a, scopeKey)
	}
	return &accessModeLevelScopeCheckerCore{
		access:             storage.Access(scope),
		globalScopeChecker: *a,
	}
}

// accessModeLevelScopeCheckerCore maintains a list of resolved roles and a desired access mode.
// TryAllowed always returns Deny, and PerformChecks does nothing.
// SubScopeChecker extracts the resource from the scope key,
// and returns a resourceLevelScopeCheckerCore with the list of resolved roles filtered down
// to those where the permission set indicates an access level for the given resource of at least the requested access level.
type accessModeLevelScopeCheckerCore struct {
	globalScopeChecker
	access storage.Access
}

func (a *accessModeLevelScopeCheckerCore) SubScopeChecker(scopeKey sac.ScopeKey) sac.ScopeCheckerCore {
	scope, ok := scopeKey.(sac.ResourceScopeKey)
	if !ok {
		return errorScopeChecker(a, scopeKey)
	}
	resource, ok := resources.MetadataForResource(permissions.Resource(scope.String()))
	if !ok {
		return sac.ErrorAccessScopeCheckerCore(errors.Wrapf(ErrUnknownResource, "on scope key %q", scopeKey))
	}
	filteredRoles := make([]permissions.ResolvedRole, 0, len(a.roles))
	for _, role := range a.roles {
		if role.GetPermissions()[string(resource.GetResource())] >= a.access {
			filteredRoles = append(filteredRoles, role)
		}
	}
	if len(filteredRoles) == 0 {
		return sac.DenyAllAccessScopeChecker()
	}

	return &resourceLevelScopeCheckerCore{
		resource: resource,
		accessModeLevelScopeCheckerCore: accessModeLevelScopeCheckerCore{
			access: a.access,
			globalScopeChecker: globalScopeChecker{
				cache: a.cache,
				roles: filteredRoles,
			},
		},
	}
}

// resourceLevelScopeCheckerCore maintains a list of resolved roles and a resource.
// TryAllowed returns Allow if the resource itself has "global" scope,
// or if there exists a resolved role where the root node is marked Included.
// SubScopeChecker extracts the cluster ID from the scope key,
// and in each effective access scope tree for a resolved role, it determines the ClustersScopeSubTree for that cluster ID.
// The list of all *ClustersScopeSubTree is filtered down to those elements which are not nil and which have a state of not Excluded.
// If this list is empty, a deny-all scope checker core is returned, otherwise a clusterLevelScopeCheckerCore is returned.
type resourceLevelScopeCheckerCore struct {
	accessModeLevelScopeCheckerCore
	resource permissions.ResourceMetadata
}

func (a *resourceLevelScopeCheckerCore) TryAllowed() sac.TryAllowedResult {
	if a.resource.GetScope() == permissions.GlobalScope {
		return sac.Allow
	}
	for _, role := range a.roles {
		scope, err := a.cache.getEffectiveAccessScope(role.GetAccessScope())
		if utils.Should(err) != nil {
			return sac.Unknown
		}
		if scope.State == sac.Included {
			return sac.Allow
		}
	}
	return sac.Deny
}

func (a *resourceLevelScopeCheckerCore) SubScopeChecker(scopeKey sac.ScopeKey) sac.ScopeCheckerCore {
	scope, ok := scopeKey.(sac.ClusterScopeKey)
	if !ok {
		return errorScopeChecker(a, scopeKey)
	}
	return &clusterNamespaceLevelScopeCheckerCore{
		clusterID:                     scope.String(),
		resourceLevelScopeCheckerCore: *a,
	}
}

type clusterNamespaceLevelScopeCheckerCore struct {
	resourceLevelScopeCheckerCore
	clusterID string
	namespace string
}

func (a *clusterNamespaceLevelScopeCheckerCore) TryAllowed() sac.TryAllowedResult {
	for _, role := range a.roles {
		scope, err := a.cache.getEffectiveAccessScope(role.GetAccessScope())
		if utils.Should(err) != nil {
			return sac.Unknown
		}
		if effectiveAccessScopeAllows(scope, a.resource, a.clusterID, a.namespace) {
			return sac.Allow
		}
	}
	return sac.Deny
}

func (a *clusterNamespaceLevelScopeCheckerCore) SubScopeChecker(scopeKey sac.ScopeKey) sac.ScopeCheckerCore {
	scope, ok := scopeKey.(sac.NamespaceScopeKey)
	if !ok {
		return errorScopeChecker(a, scopeKey)
	}
	return &clusterNamespaceLevelScopeCheckerCore{
		clusterID:                     a.clusterID,
		namespace:                     scope.String(),
		resourceLevelScopeCheckerCore: a.resourceLevelScopeCheckerCore,
	}
}

func errorScopeChecker(level interface{}, scopeKey sac.ScopeKey) sac.ScopeCheckerCore {
	return sac.ErrorAccessScopeCheckerCore(errors.Wrapf(ErrUnexpectedScopeKey, "%T scope checked encountered %q", level, scopeKey))
}

type authorizerDataCache struct {
	lock                      sync.RWMutex
	clusters                  []*storage.Cluster
	namespaces                []*storage.NamespaceMetadata
	effectiveAccessScopesByID map[string]*sac.EffectiveAccessScopeTree
}

func (c *authorizerDataCache) getEffectiveAccessScope(accessScope *storage.SimpleAccessScope) (*sac.EffectiveAccessScopeTree, error) {
	eas := c.getEffectiveAccessScopeFromCache(accessScope.GetId())
	if eas != nil {
		return eas, nil
	}

	eas, err := c.computeEffectiveAccessScope(accessScope)
	if err != nil {
		return nil, err
	}

	c.lock.Lock()
	defer c.lock.Unlock()
	// nil accessScope has empty ("") id => it will be cached as well
	c.effectiveAccessScopesByID[accessScope.GetId()] = eas

	return eas, nil
}

func (c *authorizerDataCache) getEffectiveAccessScopeFromCache(id string) *sac.EffectiveAccessScopeTree {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.effectiveAccessScopesByID[id]
}

func (c *authorizerDataCache) computeEffectiveAccessScope(accessScope *storage.SimpleAccessScope) (*sac.EffectiveAccessScopeTree, error) {
	if accessScope == nil {
		return sac.UnrestrictedEffectiveAccessScope(), nil
	}
	eas, err := sac.ComputeEffectiveAccessScope(accessScope.GetRules(), c.clusters, c.namespaces, v1.ComputeEffectiveAccessScopeRequest_MINIMAL)
	if err != nil {
		return nil, errors.Wrapf(err, "could not compute effective access scope for access scope with id %q", accessScope.GetId())
	}
	return eas, nil
}

func effectiveAccessScopeAllows(effectiveAccessScope *sac.EffectiveAccessScopeTree,
	resourceMetadata permissions.ResourceMetadata,
	clusterID, namespaceName string) bool {

	if effectiveAccessScope.State != sac.Partial {
		return effectiveAccessScope.State == sac.Included
	}

	clusterNode := effectiveAccessScope.GetClusterByID(clusterID)
	if clusterNode == nil {
		return false
	}

	if clusterNode.State == sac.Included || resourceMetadata.GetScope() == permissions.ClusterScope {
		return true
	}
	if namespaceName == "" {
		return false
	}

	namespaceNode, ok := clusterNode.Namespaces[namespaceName]

	return ok && namespaceNode.State == sac.Included
}
