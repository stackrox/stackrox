package service

import (
	"context"
	"fmt"
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

func validatePermissionSetConstrains(ps *storage.PermissionSet) error {
	if ps.GetAccessConstraint() != storage.AccessConstraint_ACCESS_CONSTRAINT_READ ||
		ps.GetScopeConstraint() != storage.ScopeConstraint_SCOPE_CONSTRAINT_CLUSTER_NAMESPACE {
		return fmt.Errorf("permission set %q must constrain access to read and resources to cluster/namespace", ps.GetName())
	}
	return nil
}

type roleManager struct {
	roleStore roleDS.DataStore
}

// createRolesForRoxClaims builds one internal role per selected permission set.
// Legacy permission requests and authentication-only requests yield a single
// role.
func (rm *roleManager) createRolesForRoxClaims(
	ctx context.Context,
	req *v1.GenerateTokenForPermissionsAndScopeRequest,
) ([]*tokens.InternalRole, error) {
	selectedSets := req.GetPermissionSets()
	if len(selectedSets) == 0 {
		return []*tokens.InternalRole{legacyRoleForRoxClaims(req.GetPermissions(), req.GetClusterScopes())}, nil
	}
	roles := make([]*tokens.InternalRole, 0, len(selectedSets))
	seen := make(map[string]struct{}, len(selectedSets))
	for _, selected := range selectedSets {
		permissionSet, err := rm.getPermissionSet(ctx, selected)
		if err != nil {
			return nil, err
		}
		if _, exists := seen[permissionSet.GetId()]; exists {
			return nil, errox.InvalidArgs.Newf("permission set %q is selected more than once", permissionSet.GetName())
		}
		seen[permissionSet.GetId()] = struct{}{}
		if err := validatePermissionSetConstrains(permissionSet); err != nil {
			return nil, errox.InvalidArgs.CausedBy(err)
		}
		role := newInternalRole(req.GetClusterScopes())
		role.Permissions = make(map[storage.Access][]string, len(permissionSet.GetResourceToAccess()))
		for resource, access := range permissionSet.GetResourceToAccess() {
			role.Permissions[access] = append(role.Permissions[access], resource)
		}
		sortPermissions(role.Permissions)
		roles = append(roles, role)
	}
	return roles, nil
}

// legacyRoleForRoxClaims builds the single role for the legacy permission map
// form, or an authentication-only role when no permissions are requested.
func legacyRoleForRoxClaims(requestedPermissions map[string]v1.Access, clusterScopes []*v1.ClusterScope) *tokens.InternalRole {
	role := newInternalRole(clusterScopes)
	if len(requestedPermissions) > 0 {
		role.Permissions = make(map[storage.Access][]string, len(requestedPermissions))
		for resource, access := range requestedPermissions {
			level := convertAccess(access)
			role.Permissions[level] = append(role.Permissions[level], resource)
		}
		sortPermissions(role.Permissions)
	}
	return role
}

// newInternalRole creates an internal role carrying the requested cluster
// scopes and no resource permissions yet.
func newInternalRole(clusterScopes []*v1.ClusterScope) *tokens.InternalRole {
	role := &tokens.InternalRole{
		RoleName: internalRoleName,
	}
	if len(clusterScopes) > 0 {
		role.Clusters = make(tokens.ClusterScopes)
	}
	for _, requestedScope := range clusterScopes {
		clusterID := requestedScope.GetClusterId()
		if requestedScope.GetFullClusterAccess() {
			role.Clusters[clusterID] = []string{"*"}
			continue
		}
		role.Clusters[clusterID] = append(role.Clusters[clusterID], requestedScope.GetNamespaces()...)
	}
	return role
}

func sortPermissions(permissions map[storage.Access][]string) {
	for _, resources := range permissions {
		slices.Sort(resources)
	}
}

func (rm *roleManager) getPermissionSet(ctx context.Context, selected *v1.SelectedPermissionSet) (*storage.PermissionSet, error) {
	if rm.roleStore == nil {
		return nil, errox.ServerError.New("Sensor proxy permission set store unavailable")
	}
	if id := selected.GetId(); id != "" {
		permissionSet, found, err := rm.roleStore.GetPermissionSet(ctx, id)
		if err != nil {
			return nil, err
		}
		if !found || permissionSet == nil {
			return nil, errox.NotFound.Newf("Sensor proxy permission set ID %q not found", id)
		}
		return permissionSet, nil
	}
	name := selected.GetName()
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
