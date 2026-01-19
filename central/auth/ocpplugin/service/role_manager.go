package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	roleDatastore "github.com/stackrox/rox/central/role/datastore"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/declarativeconfig"
)

type roleManager struct {
	roleStore roleDatastore.DataStore
}

var (
	generatedObjectTraits = &storage.Traits{Origin: storage.Traits_IMPERATIVE}
)

// createPermissionSet creates a dynamic permission set, granting the requested permissions.
// The returned information is the ID of the created permission set, or an error if any occurred
// in the creation process.
func (rm *roleManager) createPermissionSet(
	ctx context.Context,
	req *central.GenerateTokenForPermissionsAndScopeRequest,
) (string, error) {
	// TODO: Consider pruning the generated permission sets after some idle time.
	permissionSet := &storage.PermissionSet{
		ResourceToAccess: make(map[string]storage.Access),
		Traits:           generatedObjectTraits.CloneVT(),
	}
	var b strings.Builder
	readResources := req.GetReadPermissions()
	readAccessString := storage.Access_READ_ACCESS.String()
	for ix, resource := range readResources {
		if ix > 0 {
			b.WriteString(primaryListSeparator)
		}
		b.WriteString(resource)
		b.WriteString(keyValueSeparator)
		b.WriteString(readAccessString)
		permissionSet.ResourceToAccess[resource] = storage.Access_READ_ACCESS
	}
	permissionSetID := declarativeconfig.NewDeclarativePermissionSetUUID(b.String()).String()
	permissionSet.Id = permissionSetID
	permissionSet.Name = fmt.Sprintf(permissionSetNameFormat, permissionSetID)
	err := rm.roleStore.UpsertPermissionSet(ctx, permissionSet)
	if err != nil {
		return "", errors.Wrap(err, "storing permission set")
	}
	return permissionSet.GetId(), nil
}

// createAccessScope creates a dynamic access scope, granting the requested scope.
// The returned information is the identifier of the created access scope,
// or an error if any occurred in the creation process.
func (rm *roleManager) createAccessScope(
	ctx context.Context,
	req *central.GenerateTokenForPermissionsAndScopeRequest,
) (string, error) {
	// TODO: Consider pruning the generated access scopes after some idle time.
	accessScope := &storage.SimpleAccessScope{
		Description: "",
		Rules:       &storage.SimpleAccessScope_Rules{},
		Traits:      generatedObjectTraits.CloneVT(),
	}
	var b strings.Builder
	fullAccessClusters := make([]string, 0)
	clusterNamespaces := make([]*storage.SimpleAccessScope_Rules_Namespace, 0)
	for ix, clusterScope := range req.GetClusterScopes() {
		if ix > 0 {
			b.WriteString(primaryListSeparator)
		}
		b.WriteString(clusterScope.GetClusterName())
		b.WriteString(keyValueSeparator)
		if clusterScope.GetFullClusterAccess() {
			fullAccessClusters = append(fullAccessClusters, clusterScope.GetClusterName())
			b.WriteString(clusterWildCard)
		} else {
			for namespaceIndex, namespace := range clusterScope.GetNamespaces() {
				clusterNamespaces = append(clusterNamespaces, &storage.SimpleAccessScope_Rules_Namespace{
					ClusterName:   clusterScope.GetClusterName(),
					NamespaceName: namespace,
				})
				if namespaceIndex > 0 {
					b.WriteString(secondaryListSeparator)
				}
				b.WriteString(namespace)
			}
		}
	}
	accessScope.Rules.IncludedClusters = fullAccessClusters
	accessScope.Rules.IncludedNamespaces = clusterNamespaces
	accessScopeID := declarativeconfig.NewDeclarativeAccessScopeUUID(b.String()).String()
	accessScope.Id = accessScopeID
	accessScope.Name = fmt.Sprintf(accessScopeNameFormat, accessScopeID)

	err := rm.roleStore.UpsertAccessScope(ctx, accessScope)
	if err != nil {
		return "", errors.Wrap(err, "storing access scope")
	}

	return accessScope.GetId(), nil
}

// createRole creates a dynamic role, granting the requested permissions and scope.
// The returned information is the name of the created role, or an error if any occurred
// in the creation process.
func (rm *roleManager) createRole(
	ctx context.Context,
	req *central.GenerateTokenForPermissionsAndScopeRequest,
) (string, error) {
	// TODO: Consider pruning the generated roles after some idle time.
	permissionSetID, err := rm.createPermissionSet(ctx, req)
	if err != nil {
		return "", errors.Wrap(err, "creating permission set for role")
	}
	accessScopeID, err := rm.createAccessScope(ctx, req)
	if err != nil {
		return "", errors.Wrap(err, "creating access scope for role")
	}
	resultRole := &storage.Role{
		Name:            fmt.Sprintf(roleNameFormat, permissionSetID, accessScopeID),
		Description:     "Generated role for OCP console plugin",
		PermissionSetId: permissionSetID,
		AccessScopeId:   accessScopeID,
		Traits:          generatedObjectTraits.CloneVT(),
	}
	err = rm.roleStore.UpsertRole(ctx, resultRole)
	if err != nil {
		return "", errors.Wrap(err, "storing role")
	}

	return resultRole.GetName(), nil
}
