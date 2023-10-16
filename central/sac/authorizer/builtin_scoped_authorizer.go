package authorizer

import (
	"context"
	"time"

	"github.com/pkg/errors"
	clusterStore "github.com/stackrox/rox/central/cluster/datastore"
	namespaceStore "github.com/stackrox/rox/central/namespace/datastore"
	rolePkg "github.com/stackrox/rox/central/role"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/effectiveaccessscope"
	"github.com/stackrox/rox/pkg/sac/observe"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	// ErrUnexpectedScopeKey is returned when scope key does not match expected level.
	ErrUnexpectedScopeKey = errors.New("unexpected scope key")
	// ErrUnknownResource is returned when resource is unknown.
	ErrUnknownResource = errors.New("unknown resource")

	clusterMutex          sync.RWMutex
	clustersLastRefreshed time.Time
	cachedClusters        []*storage.Cluster

	namespaceMutex          sync.RWMutex
	namespacesLastRefreshed time.Time
	cachedNamespaces        []*storage.NamespaceMetadata
)

const (
	cacheRefreshPeriod = 5 * time.Second
)

// NewBuiltInScopeChecker returns a new SAC-aware scope checker for the given
// list of roles.
func NewBuiltInScopeChecker(ctx context.Context, roles []permissions.ResolvedRole) (sac.ScopeCheckerCore, error) {
	adminCtx := sac.WithGlobalAccessScopeChecker(ctx, sac.AllowAllAccessScopeChecker())

	clusters, err := fetchClusters(adminCtx)
	if err != nil {
		return nil, errors.Wrap(err, "reading all clusters")
	}
	namespaces, err := fetchNamespaces(adminCtx)
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

// globalScopeChecker maintains a list of resolved roles, a cache for
// effective access scopes, and optionally a structure for collecting traces.
//
// Allowed() always returns false since narrower scope is required to decide
// if the request should be allowed. This simplifies logic as only user with
// admin rights can be allowed on global scope.
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

func (a *globalScopeChecker) Allowed() bool {
	return false
}

func (a *globalScopeChecker) EffectiveAccessScope(resource permissions.ResourceWithAccess) (*effectiveaccessscope.ScopeTree, error) {
	return a.
		SubScopeChecker(sac.AccessModeScopeKey(resource.Access)).
		SubScopeChecker(sac.ResourceScopeKey(resource.Resource.GetResource())).
		EffectiveAccessScope(resource)
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
// maintains the access mode. It inherits Allowed() behavior.
//
// SubScopeChecker() extracts the resource from the scope key and returns a
// resourceLevelScopeCheckerCore with the list of resolved roles filtered down
// to those where the permission set indicates an access level for the given
// resource of at least the requested access level.
type accessModeLevelScopeCheckerCore struct {
	globalScopeChecker
	access storage.Access
}

func (a *accessModeLevelScopeCheckerCore) EffectiveAccessScope(resource permissions.ResourceWithAccess) (*effectiveaccessscope.ScopeTree, error) {
	if a.access < resource.Access {
		return effectiveaccessscope.DenyAllEffectiveAccessScope(), nil
	}
	return a.
		SubScopeChecker(sac.ResourceScopeKey(resource.Resource.GetResource())).
		EffectiveAccessScope(resource)
}

func (a *accessModeLevelScopeCheckerCore) SubScopeChecker(scopeKey sac.ScopeKey) sac.ScopeCheckerCore {
	scope, ok := scopeKey.(sac.ResourceScopeKey)
	if !ok {
		return errorScopeChecker(a, scopeKey)
	}
	res := permissions.Resource(scope.String())
	resource, ok := resources.MetadataForResource(res)
	if !ok {
		resource, ok = resources.MetadataForInternalResource(res)
	}
	if !ok {
		utils.Must(errors.Wrapf(ErrUnknownResource, "on scope key %q", scopeKey))
		return sac.DenyAllAccessScopeChecker()
	}
	filteredRoles := make([]permissions.ResolvedRole, 0, len(a.roles))
	for _, role := range a.roles {
		if resource.IsPermittedBy(role.GetPermissions(), a.access) {
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
// Allowed() returns true if the resource itself has "global" scope or if
// there exists a resolved role where the root node is marked as Included.
//
// SubScopeChecker() extracts the cluster ID from the scope key and returns a
// clusterNamespaceLevelScopeCheckerCore embedding the current instance and
// setting the desired cluster ID.
type resourceLevelScopeCheckerCore struct {
	accessModeLevelScopeCheckerCore
	resource permissions.ResourceMetadata
}

func (a *resourceLevelScopeCheckerCore) Allowed() bool {
	if a.resource.GetScope() == permissions.GlobalScope {
		a.trace.RecordAllowOnResourceLevel(a.access.String(), a.resource.String())
		return true
	}
	for _, role := range a.roles {
		scope, err := a.cache.getEffectiveAccessScope(role.GetAccessScope())
		if utils.ShouldErr(err) != nil {
			return false
		}
		if scope.State == effectiveaccessscope.Included {
			a.trace.RecordAllowOnResourceLevel(a.access.String(), a.resource.String())
			return true
		}
	}
	a.trace.RecordDenyOnResourceLevel(a.access.String(), a.resource.String())
	return false
}

func (a *resourceLevelScopeCheckerCore) EffectiveAccessScope(resource permissions.ResourceWithAccess) (*effectiveaccessscope.ScopeTree, error) {
	// 1. Get all roles and filter them to get only roles with desired access level (here: READ_ACCESS)
	// 2. For every role get it's effective access scope (EAS)
	// 3. Merge all EAS into a single tree
	if a.access < resource.Access {
		return effectiveaccessscope.DenyAllEffectiveAccessScope(), nil
	}
	// Ensure replaced resources are also taken into account.
	if a.resource.GetResource() != resource.Resource.GetResource() && (a.resource.ReplacingResource == nil ||
		(a.resource.ReplacingResource != nil &&
			a.resource.ReplacingResource.GetResource() != resource.Resource.GetResource())) {
		return effectiveaccessscope.DenyAllEffectiveAccessScope(), nil
	}

	if a.resource.GetScope() == permissions.GlobalScope {
		return effectiveaccessscope.UnrestrictedEffectiveAccessScope(), nil
	}

	eas := effectiveaccessscope.DenyAllEffectiveAccessScope()
	for _, role := range a.roles {
		scope, err := a.cache.getEffectiveAccessScope(role.GetAccessScope())
		if err != nil {
			return nil, err
		}
		eas.Merge(scope)
	}
	return eas, nil
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
// Allowed() returns true only if there exists a role that includes the
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

func (a *clusterNamespaceLevelScopeCheckerCore) Allowed() bool {
	for _, role := range a.roles {
		scope, err := a.cache.getEffectiveAccessScope(role.GetAccessScope())
		if utils.ShouldErr(err) != nil {
			return false
		}
		if effectiveAccessScopeAllows(scope, a.resource, a.clusterID, a.namespace) {
			a.trace.RecordAllowOnScopeLevel(a.access.String(), a.resource.String(), a.clusterID, a.namespace, role.GetRoleName())
			return true
		}
	}
	a.trace.RecordDenyOnScopeLevel(a.access.String(), a.resource.String(), a.clusterID, a.namespace)
	return false
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
	utils.Must(errors.Wrapf(ErrUnexpectedScopeKey, "%T scope checked encountered %q", level, scopeKey))
	return sac.DenyAllAccessScopeChecker()
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
	// Note: Below special handling for system scopes AccessScopeExcludeAll and AccessScopeIncludeAll scopes
	//   is replicated in central/reports/common/utils.go for access scoping vulnerability reports for reporting 2.0 feature.
	//   Vulnerability report config stores the access scope rules of the user that creates the config and uses those
	//   rules for scoping future scheduled reports. If the below behavior changes, central/reports/common/utils.go should be updated as well.
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

func fetchClusters(ctx context.Context) ([]*storage.Cluster, error) {
	clusters, valid := fetchClustersFromCache()
	if valid {
		return clusters, nil
	}

	clusters, err := fetchClustersFromDB(ctx)
	if err != nil {
		return nil, err
	}

	populateClusterCache(clusters)
	return clusters, nil
}

func fetchClustersFromCache() ([]*storage.Cluster, bool) {
	now := time.Now()

	clusterMutex.RLock()
	defer clusterMutex.RUnlock()

	if now.After(clustersLastRefreshed.Add(cacheRefreshPeriod)) {
		// The data expired, need to re-fetch and re-populate the cache
		return nil, false
	}

	result := make([]*storage.Cluster, 0, len(cachedClusters))
	result = append(result, cachedClusters...)
	return result, true
}

func fetchClustersFromDB(ctx context.Context) ([]*storage.Cluster, error) {
	return clusterStore.Singleton().GetClusters(ctx)
}

func populateClusterCache(clusters []*storage.Cluster) {
	refreshTime := time.Now()

	clusterMutex.Lock()
	defer clusterMutex.Unlock()
	if refreshTime.Before(clustersLastRefreshed.Add(cacheRefreshPeriod)) {
		return
	}

	cachedClusters = make([]*storage.Cluster, 0, len(clusters))
	for _, c := range clusters {
		stripCluster(c)
		cachedClusters = append(cachedClusters, c)
	}
	clustersLastRefreshed = refreshTime
}

func stripCluster(_ *storage.Cluster) {
	// TODO: remove any field that is not used in the SAC scope computation.
}

func fetchNamespaces(ctx context.Context) ([]*storage.NamespaceMetadata, error) {
	namespaces, valid := fetchNamespacesFromCache()
	if valid {
		return namespaces, nil
	}

	namespaces, err := fetchNamespacesFromDB(ctx)
	if err != nil {
		return nil, err
	}

	populateNamespaceCache(namespaces)
	return namespaces, nil
}

func fetchNamespacesFromCache() ([]*storage.NamespaceMetadata, bool) {
	now := time.Now()

	namespaceMutex.RLock()
	defer namespaceMutex.RUnlock()

	if now.After(namespacesLastRefreshed.Add(cacheRefreshPeriod)) {
		// The data expired, need to re-fetch and re-populate the cache
		return nil, false
	}

	result := make([]*storage.NamespaceMetadata, 0, len(cachedNamespaces))
	result = append(result, cachedNamespaces...)
	return result, true
}

func fetchNamespacesFromDB(ctx context.Context) ([]*storage.NamespaceMetadata, error) {
	return namespaceStore.Singleton().GetAllNamespaces(ctx)
}

func populateNamespaceCache(namespaces []*storage.NamespaceMetadata) {
	refreshTime := time.Now()

	namespaceMutex.Lock()
	defer namespaceMutex.Unlock()
	if refreshTime.Before(namespacesLastRefreshed.Add(cacheRefreshPeriod)) {
		return
	}

	cachedNamespaces = make([]*storage.NamespaceMetadata, 0, len(namespaces))
	for _, ns := range namespaces {
		stripNamespace(ns)
		cachedNamespaces = append(cachedNamespaces, ns)
	}
	namespacesLastRefreshed = refreshTime
}

func stripNamespace(_ *storage.NamespaceMetadata) {
	// TODO: remove any field that is not used in the SAC scope computation.
}
