package externalrolebroker

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/testutils/roletest"
	clusterviewv1alpha1 "github.com/stolostron/cluster-lifecycle-api/clusterview/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ACMClient defines the interface for interacting with the ACM clusterview API.
type ACMClient interface {
	ListUserPermissions(ctx context.Context, opts metav1.ListOptions) (*clusterviewv1alpha1.UserPermissionList, error)
	GetUserPermission(ctx context.Context, name string, opts metav1.GetOptions) (*clusterviewv1alpha1.UserPermission, error)
}

// GetResolvedRolesFromACM retrieves UserPermissions from ACM, filters them for base Kubernetes resources,
// and converts each to a permissions.ResolvedRole.
//
// The function:
//   - Calls ListUserPermissions on the ACM client
//   - Filters the results to only include UserPermissions managing base K8s resources
//   - Converts each UserPermission's ClusterRoleDefinition to a PermissionSet
//   - Converts each UserPermission's Bindings to a SimpleAccessScope
//   - Creates a ResolvedRole for each UserPermission
//
// The role name is derived from the UserPermission metadata name.
func GetResolvedRolesFromACM(ctx context.Context, client ACMClient) ([]permissions.ResolvedRole, error) {
	// Retrieve all user permissions from ACM
	userPermissionList, err := client.ListUserPermissions(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "failed to list user permissions from ACM")
	}

	// Filter to only permissions managing base Kubernetes resources
	filteredPermissions := FilterUserPermissionsForBaseK8sResources(userPermissionList.Items)

	// Convert each filtered UserPermission to a ResolvedRole
	resolvedRoles := make([]permissions.ResolvedRole, 0, len(filteredPermissions))
	for _, userPermission := range filteredPermissions {
		// Convert ClusterRoleDefinition to PermissionSet
		permissionSet := ConvertClusterRoleToPermissionSet(userPermission.Status.ClusterRoleDefinition)

		// Convert Bindings to SimpleAccessScope
		accessScope := ConvertBindingsToSimpleAccessScope(userPermission.Status.Bindings)

		// Create ResolvedRole with the UserPermission name as the role name
		resolvedRole := roletest.NewResolvedRole(
			userPermission.Name,
			permissionSet.GetResourceToAccess(),
			accessScope,
		)

		resolvedRoles = append(resolvedRoles, resolvedRole)
	}

	return resolvedRoles, nil
}
