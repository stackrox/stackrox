package service

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/pkg/errors"
	clusterDatastore "github.com/stackrox/rox/central/cluster/datastore"
	roleDatastore "github.com/stackrox/rox/central/role/datastore"
	v1 "github.com/stackrox/rox/generated/internalapi/central/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/declarativeconfig"
	"github.com/stackrox/rox/pkg/protocompat"
)

const (
	permissionSetNameFormat = "Generated permission set for %s"
	accessScopeNameFormat   = "Generated access scope for %s"
	roleNameFormat          = "Generated role for permission set %s and access scope %s"

	accessScopeDescription   = "Generated access scope for internal authentication tokens"
	permissionSetDescription = "Generated permission set for internal authentication tokens"
	roleDescription          = "Generated role for internal authentication tokens"

	primaryListSeparator   = ";"
	keyValueSeparator      = ":"
	secondaryListSeparator = ","
	clusterWildCard        = "*"
)

type roleManager struct {
	clusterStore clusterDatastore.DataStore
	roleStore    roleDatastore.DataStore
}

// generateTraitsWithExpiry creates traits for dynamically generated RBAC
// objects with an expiry time. The expiry time determines when these objects
// are eligible for pruning by the garbage collector.
func generateTraitsWithExpiry(expiresAt time.Time) (*storage.Traits, error) {
	ts, err := protocompat.ConvertTimeToTimestampOrError(expiresAt)
	return &storage.Traits{
		Origin:    storage.Traits_EPHEMERAL,
		ExpiresAt: ts,
	}, err
}

// createPermissionSet creates a dynamic permission set, granting the requested
// permissions. The returned information is the ID of the created permission
// set, or an error if any occurred in the creation process.
func (rm *roleManager) createPermissionSet(
	ctx context.Context,
	req *v1.GenerateTokenForPermissionsAndScopeRequest,
	traits *storage.Traits,
) (string, error) {
	permissionSet := &storage.PermissionSet{
		Description:      permissionSetDescription,
		ResourceToAccess: make(map[string]storage.Access),
		Traits:           traits,
	}
	var b strings.Builder
	permissions := req.GetPermissions()
	resources := make([]string, 0, len(permissions))
	for res := range permissions {
		resources = append(resources, res)
	}
	slices.Sort(resources)
	for ix, resource := range resources {
		if ix > 0 {
			b.WriteString(primaryListSeparator)
		}
		access := permissions[resource]
		accessString := access.String()
		b.WriteString(resource)
		b.WriteString(keyValueSeparator)
		b.WriteString(accessString)
		permissionSet.ResourceToAccess[resource] = convertAccess(access)
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

func convertAccess(in v1.Access) storage.Access {
	switch in {
	case v1.Access_READ_ACCESS:
		return storage.Access_READ_ACCESS
	case v1.Access_READ_WRITE_ACCESS:
		return storage.Access_READ_WRITE_ACCESS
	default:
		return storage.Access_NO_ACCESS
	}
}

// createAccessScope creates a dynamic access scope, granting the requested
// scope. The returned information is the identifier of the created access
// scope, or an error if any occurred in the creation process.
func (rm *roleManager) createAccessScope(
	ctx context.Context,
	req *v1.GenerateTokenForPermissionsAndScopeRequest,
	traits *storage.Traits,
) (string, error) {
	accessScope := &storage.SimpleAccessScope{
		Description: accessScopeDescription,
		Rules:       &storage.SimpleAccessScope_Rules{},
		Traits:      traits,
	}
	var b strings.Builder
	fullAccessClusters := make([]string, 0)
	clusterNamespaces := make([]*storage.SimpleAccessScope_Rules_Namespace, 0)
	for ix, clusterScope := range req.GetClusterScopes() {
		clusterID := clusterScope.GetClusterId()
		clusterName, found, err := rm.clusterStore.GetClusterName(ctx, clusterID)
		if err != nil {
			return "", errors.Wrap(err, "retrieving cluster name")
		}
		if !found {
			continue
		}
		if ix > 0 {
			b.WriteString(primaryListSeparator)
		}
		b.WriteString(clusterName)
		b.WriteString(keyValueSeparator)
		if clusterScope.GetFullClusterAccess() {
			fullAccessClusters = append(fullAccessClusters, clusterName)
			b.WriteString(clusterWildCard)
		} else {
			for namespaceIndex, namespace := range clusterScope.GetNamespaces() {
				clusterNamespaces = append(clusterNamespaces, &storage.SimpleAccessScope_Rules_Namespace{
					ClusterName:   clusterName,
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

// createRole creates a dynamic role, granting the requested permissions and
// scope. The returned information is the name of the created role, or an error
// if any occurred in the creation process.
func (rm *roleManager) createRole(
	ctx context.Context,
	req *v1.GenerateTokenForPermissionsAndScopeRequest,
	traits *storage.Traits,
) (string, error) {
	permissionSetID, err := rm.createPermissionSet(ctx, req, traits)
	if err != nil {
		return "", errors.Wrap(err, "creating permission set for role")
	}
	accessScopeID, err := rm.createAccessScope(ctx, req, traits)
	if err != nil {
		return "", errors.Wrap(err, "creating access scope for role")
	}
	resultRole := &storage.Role{
		Name:            fmt.Sprintf(roleNameFormat, permissionSetID, accessScopeID),
		Description:     roleDescription,
		PermissionSetId: permissionSetID,
		AccessScopeId:   accessScopeID,
		Traits:          traits,
	}
	err = rm.roleStore.UpsertRole(ctx, resultRole)
	if err != nil {
		return "", errors.Wrap(err, "storing role")
	}

	return resultRole.GetName(), nil
}
