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
)

const (
	internalRoleName = "internal role"
)

type roleManager struct {
	roleStore roleDS.DataStore
}

func (rm *roleManager) createRoleForRoxClaims(
	ctx context.Context,
	req *v1.GenerateTokenForPermissionsAndScopeRequest,
) (*tokens.InternalRole, error) {
	role := &tokens.InternalRole{
		RoleName: internalRoleName,
	}
	if req.GetPermissionSet() != nil {
		permissionSet, err := rm.getPermissionSet(ctx, req)
		if err != nil {
			return nil, err
		}
		if err := permissionsUtils.ValidateScopedReadOnlyPermissionSet(permissionSet); err != nil {
			return nil, errox.InvalidArgs.CausedBy(err)
		}
		role.Permissions = make(map[storage.Access][]string)
		for resource, access := range permissionSet.GetResourceToAccess() {
			role.Permissions[access] = append(role.Permissions[access], resource)
		}
	} else if requestedPermissions := req.GetPermissions(); len(requestedPermissions) > 0 {
		role.Permissions = make(map[storage.Access][]string)
		for resource, access := range requestedPermissions {
			level := convertAccess(access)
			role.Permissions[level] = append(role.Permissions[level], resource)
		}
	}
	for _, resources := range role.Permissions {
		slices.Sort(resources)
	}
	if len(req.GetClusterScopes()) > 0 {
		role.Clusters = make(tokens.ClusterScopes)
	}
	for _, requestedScope := range req.GetClusterScopes() {
		clusterID := requestedScope.GetClusterId()
		if requestedScope.GetFullClusterAccess() {
			role.Clusters[clusterID] = []string{"*"}
			continue
		}
		role.Clusters[clusterID] = append(role.Clusters[clusterID], requestedScope.GetNamespaces()...)
	}
	return role, nil
}

func (rm *roleManager) getPermissionSet(ctx context.Context, req *v1.GenerateTokenForPermissionsAndScopeRequest) (*storage.PermissionSet, error) {
	if err := validatePermissionSetSelector(req); err != nil {
		return nil, err
	}
	if rm.roleStore == nil {
		return nil, errox.ServerError.New("Sensor proxy permission set store unavailable")
	}
	if id := req.GetPermissionSetId(); id != "" {
		permissionSet, found, err := rm.roleStore.GetPermissionSet(ctx, id)
		if err != nil {
			return nil, err
		}
		if !found || permissionSet == nil {
			return nil, errox.NotFound.Newf("Sensor proxy permission set ID %q not found", id)
		}
		return permissionSet, nil
	}
	name := req.GetPermissionSetName()
	matches, err := rm.roleStore.GetPermissionSetsFiltered(ctx, func(ps *storage.PermissionSet) bool {
		return ps != nil && ps.GetName() == name
	})
	if err != nil {
		return nil, err
	}
	if len(matches) != 1 || matches[0] == nil {
		return nil, errox.NotFound.Newf("Sensor proxy permission set %q did not resolve uniquely", name)
	}
	return matches[0], nil
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
