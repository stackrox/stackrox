package authorizer

import (
	"context"

	"github.com/pkg/errors"
	clusterStore "github.com/stackrox/rox/central/cluster/datastore"
	namespaceStore "github.com/stackrox/rox/central/namespace/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	rolePkg "github.com/stackrox/rox/pkg/auth/role"
	"github.com/stackrox/rox/pkg/auth/role/resources"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/effectiveaccessscope"
	"github.com/stackrox/rox/pkg/sac/observe"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	// ErrUnexpectedScopeKey is returned when scope key does not match expected level.
	ErrUnexpectedScopeKey = errors.New("unexpected scope key")
	// ErrUnknownResource is returned when resource is unknown.
	ErrUnknownResource = errors.New("unknown resource")
)

// NewBuiltInScopeChecker returns a new SAC-aware scope checker for the given
// list of roles.
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

	return newGlobalScopeCheckerCore(clusters, namespaces, roles, observe.AuthzTraceFromContext(ctx)), nil
}

func newGlobalScopeCheckerCore(clusters []*storage.Cluster, namespaces []*storage.NamespaceMetadata, roles []permissions.ResolvedRole, trace *observe.AuthzTrace) sac.ScopeCheckerCore {
	scc := &globalScopeChecker{
		roles: roles,
		trace: trace,
		cache: &authorizerDataCache{
			clusters:                  clusters,
			namespaces:                namespaces,
			effectiveAccessScopesByID: make(map[string]*effectiveaccessscope.ScopeTree),
			trace:                     trace,
		},
	}

	trace.RecordKnownClustersAndNamespaces(clusters, namespaces)
	return scc
}

// globalScopeCheckerCore maintains a list of resolved roles, a cache for
// effective access scopes, and optionally a structure for collecting traces.
//
// TryAllowed() always returns Deny since narrower scope is required to decide
// if the request should be allowed. This simplifies logic as only user with
// admin rights can be allowed on global scope.
//
// PerformChecks() has nothing to do because built-in authorizer never defers
// authorization decisions, i.e., TryAllowed() returns sac.Unknown only in case
// of a non-recoverable error.
//
// SubScopeChecker() extracts the access mode from the scope key and returns
// an accessModeLevelScopeCheckerCore embedding the current instance and setting
// the desired access mode.
type globalScopeChecker struct {
	cache *authorizerDataCache

	roles []permissions.ResolvedRole

	// Should be nil unless authorization tracing is enabled for this instance.
	trace *observe.AuthzTrace
}

func (a *globalScopeChecker) TryAllowed() sac.TryAllowedResult {
	return sac.Deny
}

func (a *globalScopeChecker) PerformChecks(_ context.Context) error {
	return nil
}

func (a *globalScopeChecker) EffectiveAccessScope(resource permissions.ResourceWithAccess) (*effectiveaccessscope.ScopeTree, error) {
	return nil, errors.New("global scope checker has no effective access scope")
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

// accessModeLevelScopeCheckerCore embeds globalScopeChecker and additionally
// maintains the access mode. It inherits TryAllowed() and PerformChecks()
// behavior.
//
// SubScopeChecker() extracts the resource from the scope key and returns a
// resourceLevelScopeCheckerCore with the list of resolved roles filtered down
// to those where the permission set indicates an access level for the given
// resource of at least the requested access level.
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
		a.trace.RecordDenyOnResourceLevel(a.access.String(), resource.String())
		return sac.DenyAllAccessScopeChecker()
	}

	return &resourceLevelScopeCheckerCore{
		resource: resource,
		accessModeLevelScopeCheckerCore: accessModeLevelScopeCheckerCore{
			access: a.access,
			globalScopeChecker: globalScopeChecker{
				roles: filteredRoles,
				trace: a.trace,
				cache: a.cache,
			},
		},
	}
}

// resourceLevelScopeCheckerCore embeds accessModeLevelScopeCheckerCore and
// additionally maintains the resource.
//
// TryAllowed() returns Allow if the resource itself has "global" scope or if
// there exists a resolved role where the root node is marked as Included.
//
// SubScopeChecker() extracts the cluster ID from the scope key and returns a
// clusterNamespaceLevelScopeCheckerCore embedding the current instance and
// setting the desired cluster ID.
type resourceLevelScopeCheckerCore struct {
	accessModeLevelScopeCheckerCore
	resource permissions.ResourceMetadata
}

func (a *resourceLevelScopeCheckerCore) TryAllowed() sac.TryAllowedResult {
	if a.resource.GetScope() == permissions.GlobalScope {
		a.trace.RecordAllowOnResourceLevel(a.access.String(), a.resource.String())
		return sac.Allow
	}
	for _, role := range a.roles {
		scope, err := a.cache.getEffectiveAccessScope(role.GetAccessScope())
		if utils.Should(err) != nil {
			return sac.Unknown
		}
		if scope.State == effectiveaccessscope.Included {
			a.trace.RecordAllowOnResourceLevel(a.access.String(), a.resource.String())
			return sac.Allow
		}
	}
	a.trace.RecordDenyOnResourceLevel(a.access.String(), a.resource.String())
	return sac.Deny
}

func (a *resourceLevelScopeCheckerCore) EffectiveAccessScope(resource permissions.ResourceWithAccess) (*effectiveaccessscope.ScopeTree, error) {
	// TODO(ROX-9537): Implement it
	// 1. Get all roles and filter them to get only roles with desired access level (here: READ_ACCESS)
	// 2. For every role get it's effective access scope (EAS)
	// 3. Merge all EAS into a single tree
	panic("Implement me: ROX-9537")
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

// clusterNamespaceLevelScopeCheckerCore embeds resourceLevelScopeCheckerCore
// and maintains the cluster ID and a (potentially empty) namespace name.
//
// TryAllowed() returns Allow only if there exists a role that includes the
// requested scope.
//
// SubScopeChecker() returns another clusterNamespaceLevelScopeCheckerCore
// augmenting the current instance with the namespace name extracted from the
// scope key.
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
			a.trace.RecordAllowOnScopeLevel(a.access.String(), a.resource.String(), a.clusterID, a.namespace, role.GetRoleName())
			return sac.Allow
		}
	}
	a.trace.RecordDenyOnScopeLevel(a.access.String(), a.resource.String(), a.clusterID, a.namespace)
	return sac.Deny
}

func (a *clusterNamespaceLevelScopeCheckerCore) SubScopeChecker(scopeKey sac.ScopeKey) sac.ScopeCheckerCore {
	scope, ok := scopeKey.(sac.NamespaceScopeKey)
	if !ok || a.namespace != "" {
		// Means we are either:
		//   * on the cluster level and the scope key is not namespace,
		//   * on the namespace level where no SubScopeCheckers are possible.
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
	effectiveAccessScopesByID map[string]*effectiveaccessscope.ScopeTree

	// Should be nil unless authorization tracing is enabled for this instance.
	trace *observe.AuthzTrace
}

func (c *authorizerDataCache) getEffectiveAccessScope(accessScope *storage.SimpleAccessScope) (*effectiveaccessscope.ScopeTree, error) {
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

	c.trace.RecordEffectiveAccessScope(accessScope.GetId(), accessScope.GetName(), eas.String())

	return eas, nil
}

func (c *authorizerDataCache) getEffectiveAccessScopeFromCache(id string) *effectiveaccessscope.ScopeTree {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.effectiveAccessScopesByID[id]
}

func (c *authorizerDataCache) computeEffectiveAccessScope(accessScope *storage.SimpleAccessScope) (*effectiveaccessscope.ScopeTree, error) {
	if accessScope == nil || accessScope.Id == rolePkg.AccessScopeExcludeAll.Id {
		return effectiveaccessscope.DenyAllEffectiveAccessScope(), nil
	}
	if accessScope.Id == rolePkg.AccessScopeIncludeAll.Id {
		return effectiveaccessscope.UnrestrictedEffectiveAccessScope(), nil
	}
	eas, err := effectiveaccessscope.ComputeEffectiveAccessScope(accessScope.GetRules(), c.clusters, c.namespaces, v1.ComputeEffectiveAccessScopeRequest_MINIMAL)
	if err != nil {
		return nil, errors.Wrapf(err, "could not compute effective access scope for access scope with id %q", accessScope.GetId())
	}
	return eas, nil
}

func effectiveAccessScopeAllows(effectiveAccessScope *effectiveaccessscope.ScopeTree,
	resourceMetadata permissions.ResourceMetadata,
	clusterID, namespaceName string) bool {

	if effectiveAccessScope.State != effectiveaccessscope.Partial {
		return effectiveAccessScope.State == effectiveaccessscope.Included
	}

	clusterNode := effectiveAccessScope.GetClusterByID(clusterID)
	if clusterNode == nil {
		return false
	}

	if clusterNode.State == effectiveaccessscope.Included || resourceMetadata.GetScope() == permissions.ClusterScope {
		return true
	}
	if namespaceName == "" {
		return false
	}

	namespaceNode, ok := clusterNode.Namespaces[namespaceName]

	return ok && namespaceNode.State == effectiveaccessscope.Included
}
