package service

import (
	"context"

	"github.com/pkg/errors"
	clusterDatastore "github.com/stackrox/rox/central/cluster/datastore"
	roleDatastore "github.com/stackrox/rox/central/role/datastore"
	v1 "github.com/stackrox/rox/generated/internalapi/central/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/tokens"
)

const (
	internalRoleName = "internal role"
)

type roleManager struct {
	clusterStore clusterDatastore.DataStore
	roleStore    roleDatastore.DataStore
}

func (rm *roleManager) createRoleForRoxClaims(
	ctx context.Context,
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
		role.ClustersByName = make(tokens.ClusterScopes)
	}
	for _, requestedScope := range req.GetClusterScopes() {
		clusterName, found, err := rm.clusterStore.GetClusterName(ctx, requestedScope.GetClusterId())
		if err != nil {
			return nil, errors.Wrap(err, "getting cluster name")
		}
		if !found {
			continue
		}
		if requestedScope.GetFullClusterAccess() {
			role.ClustersByName[clusterName] = []string{"*"}
			continue
		}
		role.ClustersByName[clusterName] = append(role.ClustersByName[clusterName], requestedScope.GetNamespaces()...)
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
