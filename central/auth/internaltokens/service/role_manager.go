package service

import (
	"context"
	"slices"

	roleDS "github.com/stackrox/rox/central/role/datastore"
	v1 "github.com/stackrox/rox/generated/internalapi/central/v1"
	"github.com/stackrox/rox/generated/storage"
	permissionsUtils "github.com/stackrox/rox/pkg/auth/permissions/utils"
	"github.com/stackrox/rox/pkg/auth/tokens"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/uuid"
)

const (
	internalRoleName = "internal role"
)

type roleManager struct {
	roleStore roleDS.DataStore
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

	if len(req.GetPermissionSets()) == 0 {
		return nil, errox.InvalidArgs.New("Sensor proxy permission sets are required")
	}
	var names, ids []string
	for _, grant := range req.GetPermissionSets() {
		if grant.GetName() != "" {
			names = append(names, grant.GetName())
		} else {
			ids = append(ids, grant.GetId())
		}
	}
	permissionSets, err := rm.getPermissionSetsByNames(ctx, names)
	if err != nil {
		return nil, err
	}
	byID, err := rm.getPermissionSetsByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	for _, id := range ids {
		permissionSets = append(permissionSets, byID[id])
	}
	for _, permissionSet := range permissionSets {
		if err := permissionsUtils.ValidateScopedReadOnlyPermissionSet(permissionSet); err != nil {
			return nil, errox.InvalidArgs.CausedBy(err)
		}
	}
	return &tokens.InternalRole{
		RoleName:    internalRoleName,
		Clusters:    clusterScopesFromRequest(req.GetClusterScopes()),
		Permissions: permissionsFromSets(permissionSets),
	}, nil
}

func (rm *roleManager) getPermissionSetsByNames(ctx context.Context, permissionSetNames []string) ([]*storage.PermissionSet, error) {
	if len(permissionSetNames) == 0 {
		return nil, nil
	}
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

func (rm *roleManager) getPermissionSetsByIDs(ctx context.Context, permissionSetIDs []string) (map[string]*storage.PermissionSet, error) {
	sets := make(map[string]*storage.PermissionSet, len(permissionSetIDs))
	for _, id := range permissionSetIDs {
		if _, err := uuid.FromString(id); err != nil {
			return nil, errox.InvalidArgs.Newf("invalid Sensor proxy permission set ID %q", id)
		}
		permissionSet, found, err := rm.roleStore.GetPermissionSet(ctx, id)
		if err != nil {
			return nil, err
		}
		if !found {
			return nil, errox.NotFound.Newf("Sensor proxy permission set ID %q not found", id)
		}
		sets[id] = permissionSet
	}
	return sets, nil
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
