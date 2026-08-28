package externalrolebroker

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/auth/tokens"
	clusterviewv1alpha1 "github.com/stolostron/cluster-lifecycle-api/clusterview/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// acmClient is the consumer-side interface for acm.Client, scoped to what this package uses.
type acmClient interface {
	ListUserPermissions(ctx context.Context, opts metav1.ListOptions) (*clusterviewv1alpha1.UserPermissionList, error)
}

// GetResolvedRolesFromACM retrieves UserPermissions from ACM, filters them for supported Kubernetes resources,
// and converts each to a permissions.ResolvedRole.
//
// The function:
//   - Calls ListUserPermissions on the ACM client
//   - Filters the results to only include UserPermissions managing supported K8s resources
//   - Converts each UserPermission's ClusterRoleDefinition to a PermissionSet
//   - Converts each UserPermission's Bindings to a SimpleAccessScope
//   - Creates a ResolvedRole for each UserPermission
//
// The role name is derived from the UserPermission metadata name
func GetResolvedRolesFromACM(
	ctx context.Context,
	client acmClient,
	clusterIDResolver tokens.ClusterResolver,
) ([]*tokens.InternalRole, error) {
	// Retrieve all user permissions from ACM
	userPermissionList, err := client.ListUserPermissions(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "failed to list user permissions")
	}
	if userPermissionList == nil {
		return nil, errors.New("received nil user permission list from ACM")
	}

	// Filter to only permissions managing supported Kubernetes resources
	filteredPermissions := FilterUserPermissionsForSupportedK8sResources(userPermissionList.Items)

	// Convert each filtered UserPermission to a ResolvedRole
	resolvedRoles := make([]*tokens.InternalRole, 0, len(filteredPermissions))
	for _, userPermission := range filteredPermissions {
		// Convert ClusterRoleDefinition to PermissionSet
		permissionSet := ConvertClusterRoleToPermissionSet(userPermission.Status.ClusterRoleDefinition)

		// Convert Bindings to SimpleAccessScope
		accessScope := ConvertBindingsToSimpleAccessScope(userPermission.Status.Bindings)

		resolvedRole, roleErr := tokens.NewInternalRoleFromPermissionsAndScope(
			ctx,
			userPermission.Name,
			permissionSet,
			accessScope,
			clusterIDResolver,
		)
		if roleErr != nil {
			return nil, errors.Wrap(roleErr, "failed to create role")
		}
		resolvedRoles = append(resolvedRoles, resolvedRole)
	}
	return resolvedRoles, nil
}
