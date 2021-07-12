package authorizer

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/default-authz-plugin/pkg/payload"
	clusterStore "github.com/stackrox/rox/central/cluster/datastore"
	namespaceStore "github.com/stackrox/rox/central/namespace/datastore"
	roleStore "github.com/stackrox/rox/central/role/datastore"
	"github.com/stackrox/rox/central/role/resources"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/client"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	initClientOnce sync.Once

	singletonClient client.Client

	view = sac.AccessModeScopeKey(storage.Access_READ_ACCESS).Verb()
	edit = sac.AccessModeScopeKey(storage.Access_READ_WRITE_ACCESS).Verb()

	log = logging.LoggerForModule()
)

type builtInScopeAuthorizer struct {
	clusterStore   clusterStore.DataStore
	namespaceStore namespaceStore.DataStore
	roleStore      roleStore.DataStore
}

// Singleton provides the interface for non-service external interaction.
func Singleton() client.Client {
	initClientOnce.Do(func() {
		singletonClient = &builtInScopeAuthorizer{
			clusterStore:   clusterStore.Singleton(),
			namespaceStore: namespaceStore.Singleton(),
			roleStore:      roleStore.Singleton(),
		}
	})
	return singletonClient
}

type authorizerDataCache struct {
	clusters                  []*storage.Cluster
	namespaces                []*storage.NamespaceMetadata
	effectiveAccessScopesByID map[string]*sac.EffectiveAccessScopeTree
}

// ForUser returns a list of allowed scopes, a list of denied scopes, and any errors.
//TODO(ROX-7392): Pass Identity with Principal
func (a *builtInScopeAuthorizer) ForUser(ctx context.Context, principal payload.Principal, scopes ...payload.AccessScope) ([]payload.AccessScope, []payload.AccessScope, error) {
	adminCtx := sac.WithGlobalAccessScopeChecker(ctx, sac.AllowAllAccessScopeChecker())

	roles, err := a.resolveRoles(adminCtx, principal.Roles...)
	if err != nil {
		return nil, nil, err
	}
	clusters, err := a.clusterStore.GetClusters(adminCtx)
	if err != nil {
		return nil, nil, err
	}
	namespaces, err := a.namespaceStore.GetNamespaces(adminCtx)
	if err != nil {
		return nil, nil, err
	}

	cache := &authorizerDataCache{
		clusters:                  clusters,
		namespaces:                namespaces,
		effectiveAccessScopesByID: make(map[string]*sac.EffectiveAccessScopeTree),
	}

	var denied []payload.AccessScope
	var allowed []payload.AccessScope

SCOPE:
	for _, scope := range scopes {
		for _, role := range roles {
			permitted, err := roleHasPermissions(role, cache, scope)
			if err != nil {
				return nil, nil, err
			}
			if permitted {
				allowed = append(allowed, scope)
				continue SCOPE
			}
		}
		denied = append(denied, scope)
	}

	return allowed, denied, nil
}

func (a *builtInScopeAuthorizer) resolveRoles(ctx context.Context, roleNames ...string) (map[string]*permissions.ResolvedRole, error) {
	roles := make(map[string]*permissions.ResolvedRole, len(roleNames))
	for _, roleName := range roleNames {
		if _, ok := roles[roleName]; ok {
			continue
		}
		role, err := a.roleStore.GetAndResolveRole(ctx, roleName)
		if err != nil {
			return nil, errors.Errorf("role with name %q does not exist", roleName)
		}
		roles[roleName] = role
	}
	return roles, nil
}

func (c *authorizerDataCache) getEffectiveAccessScope(accessScope *storage.SimpleAccessScope) (*sac.EffectiveAccessScopeTree, error) {
	if eas, ok := c.effectiveAccessScopesByID[accessScope.GetId()]; ok {
		return eas, nil
	}
	eas, err := c.computeEffectiveAccessScope(accessScope)
	if err != nil {
		return nil, err
	}

	c.effectiveAccessScopesByID[accessScope.GetId()] = eas

	return eas, nil
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

func roleHasPermissions(r *permissions.ResolvedRole, cache *authorizerDataCache, scope payload.AccessScope) (bool, error) {
	if !permissionsSetAllows(r.PermissionSet, scope) {
		return false, nil
	}

	effectiveAccessScope, err := cache.getEffectiveAccessScope(r.AccessScope)
	if err != nil {
		return false, errors.Wrapf(err, "could not get effective access scope with id %q for role %q", r.AccessScope.GetId(), r.Role.GetName())
	}

	return effectiveAccessScopeAllows(effectiveAccessScope, scope), nil

}

func permissionsSetAllows(rolePermissionSet *storage.PermissionSet, scope payload.AccessScope) bool {
	queriedAccessScope := storage.Access_NO_ACCESS
	switch scope.Verb {
	case view:
		queriedAccessScope = storage.Access_READ_ACCESS
	case "", edit:
		queriedAccessScope = storage.Access_READ_WRITE_ACCESS
	}

	if scope.Noun != "" {
		return rolePermissionSet.ResourceToAccess[scope.Noun] >= queriedAccessScope
	}

	// If there is no noun let's assume query applies for all resources
	for _, rwa := range resources.ListAll() {
		if rolePermissionSet.ResourceToAccess[string(rwa)] < queriedAccessScope {
			return false
		}
	}

	return true
}

func effectiveAccessScopeAllows(effectiveAccessScope *sac.EffectiveAccessScopeTree, scope payload.AccessScope) bool {
	if effectiveAccessScope.State != sac.Partial {
		return effectiveAccessScope.State == sac.Included
	}

	clusterName := scope.Attributes.Cluster.Name
	namespaceName := scope.Attributes.Namespace
	resourceMetadata, _ := resources.MetadataForResource(permissions.Resource(scope.Noun))

	if resourceMetadata.GetScope() == permissions.GlobalScope {
		return true
	}

	clusterNode, ok := effectiveAccessScope.Clusters[clusterName]
	if !ok {
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
