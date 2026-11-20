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

func (rm *roleManager) createScopedRoles(
	ctx context.Context, req *v1.GenerateTokenForPermissionsAndScopeRequest,
) ([]*tokens.InternalRole, error) {
	if rm.roleStore == nil {
		return nil, errox.ServerError.New("Sensor proxy permission set store unavailable")
	}
	if len(req.GetPermissionSets()) == 0 {
		return nil, errox.InvalidArgs.New("Sensor proxy permission sets are required")
	}
	names := make([]string, 0, len(req.GetPermissionSets()))
	ids := make([]string, 0, len(req.GetPermissionSets()))
	for _, grant := range req.GetPermissionSets() {
		if grant.GetName() != "" {
			names = append(names, grant.GetName())
		} else {
			ids = append(ids, grant.GetId())
		}
	}
	sets, err := rm.getPermissionSetsByNames(ctx, names)
	if err != nil {
		return nil, err
	}
	byID, err := rm.getPermissionSetsByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	byName := make(map[string]*storage.PermissionSet, len(sets))
	for _, permissionSet := range sets {
		byName[permissionSet.GetName()] = permissionSet
	}
	seenIDs := make(map[string]struct{}, len(req.GetPermissionSets()))
	roles := make([]*tokens.InternalRole, 0, len(req.GetPermissionSets()))
	clusters := clusterScopesFromRequest(req.GetClusterScopes())
	for _, grant := range req.GetPermissionSets() {
		permissionSet := byName[grant.GetName()]
		if grant.GetId() != "" {
			permissionSet = byID[grant.GetId()]
		}
		key := permissionSet.GetId()
		if key == "" {
			key = permissionSet.GetName()
		}
		if _, exists := seenIDs[key]; exists {
			return nil, errox.InvalidArgs.Newf("duplicate per-access permission set %q", permissionSet.GetName())
		}
		seenIDs[key] = struct{}{}
		if err := permissionsUtils.ValidateScopedReadOnlyPermissionSet(permissionSet); err != nil {
			return nil, errox.InvalidArgs.CausedBy(err)
		}
		// The policy validated each scope before the permission sets were loaded.
		scopes, err := tokens.HTTPRequestScopesFromProto(grant.GetHttpRequestScopes())
		if err != nil {
			return nil, errox.InvalidArgs.CausedBy(err)
		}
		roles = append(roles, &tokens.InternalRole{
			RoleName:          internalRoleName,
			Permissions:       permissionsFromSet(permissionSet),
			Clusters:          clusters,
			HTTPRequestScopes: scopes,
		})
	}
	return roles, nil
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

func permissionsFromSet(permissionSet *storage.PermissionSet) map[storage.Access][]string {
	permissions := make(map[storage.Access][]string)
	for resource, access := range permissionSet.GetResourceToAccess() {
		permissions[access] = append(permissions[access], resource)
	}
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
