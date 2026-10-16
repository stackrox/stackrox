package service

import (
	"context"

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

func (rm *roleManager) createSensorConsoleRole(
	ctx context.Context, roles roleDS.DataStore, name string, scopes []*v1.ClusterScope,
) (*tokens.InternalRole, error) {
	return rm.createSensorRole(ctx, roles, name, clusterScopesFromRequest(scopes))
}

func (rm *roleManager) createSensorRole(
	ctx context.Context, roles roleDS.DataStore, name string, scopes tokens.ClusterScopes,
) (*tokens.InternalRole, error) {
	if roles == nil {
		return nil, errox.NotAuthorized.New("Sensor proxy role store unavailable")
	}
	resolved, err := roles.GetAndResolveRole(ctx, name)
	if err != nil {
		return nil, err
	}
	if resolved == nil {
		return nil, errox.NotFound.Newf("Sensor proxy role %q not found", name)
	}

	role := &tokens.InternalRole{
		RoleName:    name,
		Clusters:    scopes,
		Permissions: make(map[storage.Access][]string),
	}
	for resource, access := range resolved.GetPermissions() {
		switch access {
		case storage.Access_NO_ACCESS:
		case storage.Access_READ_ACCESS:
			role.Permissions[storage.Access_READ_ACCESS] = append(role.Permissions[storage.Access_READ_ACCESS], resource)
		default:
			return nil, errox.InvalidArgs.Newf("Sensor proxy role must grant only read permissions; %q has %q", resource, access)
		}
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
