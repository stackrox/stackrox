package service

import (
	"context"
	"slices"

	roleDS "github.com/stackrox/rox/central/role/datastore"
	v1 "github.com/stackrox/rox/generated/internalapi/central/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/tokens"
	"github.com/stackrox/rox/pkg/errox"
)

const (
	internalRoleName = "internal role"
)

type roleManager struct{}

func (rm *roleManager) createSensorRole(
	ctx context.Context, roleStore roleDS.DataStore, permissionSetIDs []string, name string, scopes tokens.ClusterScopes,
) (*tokens.InternalRole, error) {
	if len(permissionSetIDs) == 0 {
		return nil, errox.InvalidArgs.New("Sensor proxy permission sets are required")
	}
	if roleStore == nil {
		return nil, errox.NotAuthorized.New("Sensor proxy permission set store unavailable")
	}
	role := &tokens.InternalRole{
		RoleName:    name,
		Clusters:    scopes,
		Permissions: make(map[storage.Access][]string),
	}
	permissions := make(map[string]storage.Access)
	for _, permissionSetID := range permissionSetIDs {
		permissionSet, found, err := roleStore.GetPermissionSet(ctx, permissionSetID)
		if err != nil {
			return nil, err
		}
		if !found || permissionSet == nil {
			return nil, errox.NotFound.Newf("Sensor proxy permission set %q not found", permissionSetID)
		}
		for resource, access := range permissionSet.GetResourceToAccess() {
			if current, found := permissions[resource]; !found || access > current {
				permissions[resource] = access
			}
		}
	}
	for resource, access := range permissions {
		role.Permissions[access] = append(role.Permissions[access], resource)
	}
	for _, resources := range role.Permissions {
		slices.Sort(resources)
	}
	return role, nil
}

func (rm *roleManager) createRoleForRoxClaims(
	_ context.Context,
	req *v1.GenerateTokenForPermissionsAndScopeRequest,
) (*tokens.InternalRole, error) {
	role := &tokens.InternalRole{
		RoleName: internalRoleName,
	}
	requestedPermissions := req.GetPermissions()
	if len(requestedPermissions) > 0 {
		role.Permissions = make(map[storage.Access][]string)
	}
	for resource, v1Access := range req.GetPermissions() {
		access := convertAccess(v1Access)
		if _, found := role.Permissions[access]; !found {
			role.Permissions[access] = make([]string, 0)
		}
		role.Permissions[access] = append(role.Permissions[access], resource)
	}
	role.Clusters = clusterScopesFromRequest(req.GetClusterScopes())
	return role, nil
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
