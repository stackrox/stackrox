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

// createSensorMetricsRole takes permissions from the Central-managed role, but
// never its static access scope. The token is limited to the authenticated
// Sensor's cluster rather than a scope chosen by the proxy caller.
func (rm *roleManager) createSensorMetricsRole(
	ctx context.Context, roles roleDS.DataStore, name, sensorClusterID string,
) (*tokens.InternalRole, error) {
	if roles == nil || sensorClusterID == "" {
		return nil, errox.NotAuthorized.New("metrics role store or sensor cluster ID unavailable")
	}
	resolved, err := roles.GetAndResolveRole(ctx, name)
	if err != nil {
		return nil, err
	}
	if resolved == nil {
		return nil, errox.NotFound.Newf("secured-cluster metrics role %q not found", name)
	}

	role := &tokens.InternalRole{
		RoleName: name,
		Clusters: tokens.ClusterScopes{
			sensorClusterID: {"*"},
		},
		Permissions: make(map[storage.Access][]string),
	}
	for resource, access := range resolved.GetPermissions() {
		switch access {
		case storage.Access_NO_ACCESS:
		case storage.Access_READ_ACCESS:
			role.Permissions[storage.Access_READ_ACCESS] = append(role.Permissions[storage.Access_READ_ACCESS], resource)
		default:
			return nil, errox.InvalidArgs.Newf("metrics role must grant only read permissions; %q has %q", resource, access)
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
