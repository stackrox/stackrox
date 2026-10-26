package service

import (
	"context"
	"slices"

	roleDS "github.com/stackrox/rox/central/role/datastore"
	v1 "github.com/stackrox/rox/generated/internalapi/central/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/tokens"
	"github.com/stackrox/rox/pkg/centralproxy"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/grpc/authn"
)

const (
	internalRoleName = "internal role"
)

type roleManager struct {
	roleStore roleDS.DataStore
}

func validateSensorMetricsRole(role *tokens.InternalRole) error {
	for access := range role.Permissions {
		if access != storage.Access_NO_ACCESS && access != storage.Access_READ_ACCESS {
			return errox.InvalidArgs.New("Sensor metrics permission set grants non-read-only access")
		}
	}
	return nil
}

func (rm *roleManager) createRoleFromPermissionSets(
	ctx context.Context, req *v1.GenerateTokenForPermissionsAndScopeRequest,
) (*tokens.InternalRole, error) {
	role, err := rm.createSensorRole(ctx, req)
	if err != nil {
		return nil, err
	}
	if slices.Contains(req.GetPermissionSetNames(), centralproxy.MetricsPermissionSetName) {
		role.Clusters = tokens.ClusterScopes{authn.IdentityFromContextOrNil(ctx).Service().GetId(): {"*"}}
		if err := validateSensorMetricsRole(role); err != nil {
			return nil, err
		}
	}
	return role, nil
}
func (rm *roleManager) createRoleForRoxClaims(
	_ context.Context,
	req *v1.GenerateTokenForPermissionsAndScopeRequest,
) (*tokens.InternalRole, error) {
	return &tokens.InternalRole{
		RoleName:    internalRoleName,
		Permissions: permissionsFromRequest(req.GetPermissions()),
		Clusters:    clusterScopesFromRequest(req.GetClusterScopes()),
	}, nil
}

func (rm *roleManager) createSensorRole(
	ctx context.Context, req *v1.GenerateTokenForPermissionsAndScopeRequest,
) (*tokens.InternalRole, error) {
	if rm.roleStore == nil {
		return nil, errox.ServerError.New("Sensor proxy permission set store unavailable")
	}

	permissionSetNames := req.GetPermissionSetNames()
	if len(permissionSetNames) == 0 {
		return nil, errox.InvalidArgs.New("Sensor proxy permission sets are required")
	}
	permissionSets, err := rm.getPermissionSetsByNames(ctx, permissionSetNames)
	if err != nil {
		return nil, err
	}
	for _, permissionSet := range permissionSets {
		if err := validateSensorProxyPermissionSet(permissionSet); err != nil {
			return nil, err
		}
	}
	return &tokens.InternalRole{
		RoleName:    internalRoleName,
		Clusters:    clusterScopesFromRequest(req.GetClusterScopes()),
		Permissions: permissionsFromSets(permissionSets),
	}, nil
}

func (rm *roleManager) createScopedRoles(
	ctx context.Context, req *v1.GenerateTokenForPermissionsAndScopeRequest,
) ([]*tokens.InternalRole, error) {
	if rm.roleStore == nil {
		return nil, errox.ServerError.New("Sensor proxy permission set store unavailable")
	}
	names := make([]string, 0, len(req.GetScopedPermissionSets()))
	for _, grant := range req.GetScopedPermissionSets() {
		names = append(names, grant.GetName())
	}
	sets, err := rm.getPermissionSetsByNames(ctx, names)
	if err != nil {
		return nil, err
	}
	byName := make(map[string]*storage.PermissionSet, len(sets))
	for _, permissionSet := range sets {
		if err := validateSensorProxyPermissionSet(permissionSet); err != nil {
			return nil, err
		}
		byName[permissionSet.GetName()] = permissionSet
	}
	roles := make([]*tokens.InternalRole, 0, len(names))
	for _, grant := range req.GetScopedPermissionSets() {
		// The policy validated each scope before the permission sets were loaded.
		scopes, err := tokens.HTTPRequestScopesFromProto(grant.GetHttpRequestScopes())
		if err != nil {
			return nil, errox.InvalidArgs.CausedBy(err)
		}
		roles = append(roles, &tokens.InternalRole{
			RoleName:          internalRoleName,
			Permissions:       permissionsFromSets([]*storage.PermissionSet{byName[grant.GetName()]}),
			HTTPRequestScopes: scopes,
		})
	}
	return roles, nil
}

func validateSensorProxyPermissionSet(permissionSet *storage.PermissionSet) error {
	if permissionSet.GetAccessConstraint() != storage.AccessConstraint_ACCESS_CONSTRAINT_READ ||
		permissionSet.GetScopeConstraint() != storage.ScopeConstraint_SCOPE_CONSTRAINT_CLUSTER_NAMESPACE {
		return errox.InvalidArgs.Newf("Sensor proxy permission set %q must constrain access to read and resources to cluster/namespace",
			permissionSet.GetName())
	}
	return nil
}

func (rm *roleManager) getPermissionSetsByNames(ctx context.Context, permissionSetNames []string) ([]*storage.PermissionSet, error) {
	matches, err := rm.roleStore.GetPermissionSetsFiltered(ctx, func(ps *storage.PermissionSet) bool {
		return slices.Contains(permissionSetNames, ps.GetName())
	})
	if err != nil {
		return nil, err
	}
	// Check if all requested permission sets have been found.
	for _, name := range permissionSetNames {
		found := false
		for _, ps := range matches {
			if name == ps.GetName() {
				found = true
			}
		}
		if !found {
			return nil, errox.NotFound.Newf("Sensor proxy permission set %q not found", name)
		}
	}
	return matches, nil
}

func permissionsFromSets(permissionSets []*storage.PermissionSet) map[storage.Access][]string {
	// Pick the highest access to a resource from all matches.
	resourceAccess := make(map[string]storage.Access)
	for _, ps := range permissionSets {
		for resource, access := range ps.GetResourceToAccess() {
			if current, found := resourceAccess[resource]; !found || access > current {
				resourceAccess[resource] = access
			}
		}
	}
	// Collect the resources per access.
	permissions := make(map[storage.Access][]string)
	for resource, access := range resourceAccess {
		permissions[access] = append(permissions[access], resource)
	}
	// Sort the resources by name.
	for _, resources := range permissions {
		slices.Sort(resources)
	}
	return permissions
}

func permissionsFromRequest(requestedPermissions map[string]v1.Access) map[storage.Access][]string {
	if len(requestedPermissions) == 0 {
		return nil
	}
	permissions := make(map[storage.Access][]string)
	for resource, v1Access := range requestedPermissions {
		access := convertAccess(v1Access)
		if _, found := permissions[access]; !found {
			permissions[access] = make([]string, 0)
		}
		permissions[access] = append(permissions[access], resource)
	}
	return permissions
}

func clusterScopesFromRequest(scopes []*v1.ClusterScope) tokens.ClusterScopes {
	if len(scopes) == 0 {
		return nil
	}
	clusters := make(tokens.ClusterScopes)
	for _, requestedScope := range scopes {
		clusterID := requestedScope.GetClusterId()
		if requestedScope.GetFullClusterAccess() {
			clusters[clusterID] = []string{"*"}
			continue
		}
		clusters[clusterID] = append(clusters[clusterID], requestedScope.GetNamespaces()...)
	}
	return clusters
}

func convertAccess(access v1.Access) storage.Access {
	switch access {
	case v1.Access_READ_ACCESS:
		return storage.Access_READ_ACCESS
	case v1.Access_READ_WRITE_ACCESS:
		return storage.Access_READ_WRITE_ACCESS
	default:
		return storage.Access_NO_ACCESS
	}
}
