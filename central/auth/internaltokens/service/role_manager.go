package service

import (
	"context"

	"github.com/pkg/errors"
	clusterDatastore "github.com/stackrox/rox/central/cluster/datastore"
	roleDatastore "github.com/stackrox/rox/central/role/datastore"
	v1 "github.com/stackrox/rox/generated/internalapi/central/v1"
	"github.com/stackrox/rox/pkg/auth/tokens"
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
		Permissions: make(map[string]string),
	}
	for resource, access := range req.GetPermissions() {
		role.Permissions[resource] = access.String()
	}
	for _, requestedScope := range req.GetClusterScopes() {
		clusterName, found, err := rm.clusterStore.GetClusterName(ctx, requestedScope.GetClusterId())
		if err != nil {
			return nil, errors.Wrap(err, "getting cluster name")
		}
		if !found {
			continue
		}
		clusterScope := &tokens.ClusterScope{
			ClusterName:       clusterName,
			ClusterFullAccess: requestedScope.GetFullClusterAccess(),
			Namespaces:        requestedScope.GetNamespaces(),
		}
		role.ClusterScopes = append(role.ClusterScopes, clusterScope)
	}
	return role, nil
}
