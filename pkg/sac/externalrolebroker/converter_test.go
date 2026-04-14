package externalrolebroker

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac/resources"
	clusterviewv1alpha1 "github.com/stolostron/cluster-lifecycle-api/clusterview/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	rbacv1 "k8s.io/api/rbac/v1"
)

func TestConvertClusterRoleToPermissionSet(t *testing.T) {
	tests := map[string]struct {
		clusterRoleDef      clusterviewv1alpha1.ClusterRoleDefinition
		expectedPermissions map[string]storage.Access
		unexpectedResources []string
		minPermissionCount  int
	}{
		"read-only namespace access": {
			clusterRoleDef: clusterviewv1alpha1.ClusterRoleDefinition{
				Rules: []rbacv1.PolicyRule{
					{
						APIGroups: []string{""},
						Resources: []string{"namespaces"},
						Verbs:     []string{"get", "list", "watch"},
					},
				},
			},
			expectedPermissions: map[string]storage.Access{
				string(resources.Namespace.GetResource()): storage.Access_READ_ACCESS,
			},
		},
		"write access to secrets": {
			clusterRoleDef: clusterviewv1alpha1.ClusterRoleDefinition{
				Rules: []rbacv1.PolicyRule{
					{
						APIGroups: []string{""},
						Resources: []string{"secrets"},
						Verbs:     []string{"create", "update", "delete"},
					},
				},
			},
			expectedPermissions: map[string]storage.Access{
				string(resources.Secret.GetResource()): storage.Access_READ_WRITE_ACCESS,
			},
		},
		"read and write access to roles": {
			clusterRoleDef: clusterviewv1alpha1.ClusterRoleDefinition{
				Rules: []rbacv1.PolicyRule{
					{
						APIGroups: []string{"rbac.authorization.k8s.io"},
						Resources: []string{"roles"},
						Verbs:     []string{"get", "list", "create", "update"},
					},
				},
			},
			expectedPermissions: map[string]storage.Access{
				string(resources.K8sRole.GetResource()): storage.Access_READ_WRITE_ACCESS,
			},
		},
		"multiple resources with different access levels": {
			clusterRoleDef: clusterviewv1alpha1.ClusterRoleDefinition{
				Rules: []rbacv1.PolicyRule{
					{
						APIGroups: []string{""},
						Resources: []string{"namespaces"},
						Verbs:     []string{"get", "list"},
					},
					{
						APIGroups: []string{""},
						Resources: []string{"secrets"},
						Verbs:     []string{"create", "delete"},
					},
					{
						APIGroups: []string{"rbac.authorization.k8s.io"},
						Resources: []string{"roles", "rolebindings"},
						Verbs:     []string{"get", "list", "watch"},
					},
				},
			},
			expectedPermissions: map[string]storage.Access{
				string(resources.Namespace.GetResource()):      storage.Access_READ_ACCESS,
				string(resources.Secret.GetResource()):         storage.Access_READ_WRITE_ACCESS,
				string(resources.K8sRole.GetResource()):        storage.Access_READ_ACCESS,
				string(resources.K8sRoleBinding.GetResource()): storage.Access_READ_ACCESS,
			},
		},
		"wildcard verbs grant full access": {
			clusterRoleDef: clusterviewv1alpha1.ClusterRoleDefinition{
				Rules: []rbacv1.PolicyRule{
					{
						APIGroups: []string{""},
						Resources: []string{"secrets"},
						Verbs:     []string{"*"},
					},
				},
			},
			expectedPermissions: map[string]storage.Access{
				string(resources.Secret.GetResource()): storage.Access_READ_WRITE_ACCESS,
			},
		},
		"wildcard resources grant access to all base resources": {
			clusterRoleDef: clusterviewv1alpha1.ClusterRoleDefinition{
				Rules: []rbacv1.PolicyRule{
					{
						APIGroups: []string{"*"},
						Resources: []string{"*"},
						Verbs:     []string{"get", "list"},
					},
				},
			},
			minPermissionCount: 5, // All 5 base resources
			expectedPermissions: map[string]storage.Access{
				string(resources.Namespace.GetResource()):      storage.Access_READ_ACCESS,
				string(resources.Secret.GetResource()):         storage.Access_READ_ACCESS,
				string(resources.ServiceAccount.GetResource()): storage.Access_READ_ACCESS,
				string(resources.K8sRole.GetResource()):        storage.Access_READ_ACCESS,
				string(resources.K8sRoleBinding.GetResource()): storage.Access_READ_ACCESS,
			},
		},
		"subresource access": {
			clusterRoleDef: clusterviewv1alpha1.ClusterRoleDefinition{
				Rules: []rbacv1.PolicyRule{
					{
						APIGroups: []string{""},
						Resources: []string{"secrets/status"},
						Verbs:     []string{"get", "update"},
					},
				},
			},
			expectedPermissions: map[string]storage.Access{
				string(resources.Secret.GetResource()): storage.Access_READ_WRITE_ACCESS,
			},
		},
		"non-base resources are ignored": {
			clusterRoleDef: clusterviewv1alpha1.ClusterRoleDefinition{
				Rules: []rbacv1.PolicyRule{
					{
						APIGroups: []string{"apps"},
						Resources: []string{"deployments", "statefulsets"},
						Verbs:     []string{"get", "list", "create"},
					},
					{
						APIGroups: []string{""},
						Resources: []string{"pods", "configmaps"},
						Verbs:     []string{"get", "list"},
					},
				},
			},
			expectedPermissions: map[string]storage.Access{},
		},
		"mixed base and non-base resources": {
			clusterRoleDef: clusterviewv1alpha1.ClusterRoleDefinition{
				Rules: []rbacv1.PolicyRule{
					{
						APIGroups: []string{""},
						Resources: []string{"pods", "secrets", "configmaps"},
						Verbs:     []string{"get", "list"},
					},
				},
			},
			expectedPermissions: map[string]storage.Access{
				string(resources.Secret.GetResource()): storage.Access_READ_ACCESS,
			},
			unexpectedResources: []string{"Pod", "ConfigMap"},
		},
		"escalating permissions on same resource": {
			clusterRoleDef: clusterviewv1alpha1.ClusterRoleDefinition{
				Rules: []rbacv1.PolicyRule{
					{
						APIGroups: []string{""},
						Resources: []string{"secrets"},
						Verbs:     []string{"get", "list"},
					},
					{
						APIGroups: []string{""},
						Resources: []string{"secrets"},
						Verbs:     []string{"create", "delete"},
					},
				},
			},
			expectedPermissions: map[string]storage.Access{
				string(resources.Secret.GetResource()): storage.Access_READ_WRITE_ACCESS,
			},
		},
		"all base resources with full access": {
			clusterRoleDef: clusterviewv1alpha1.ClusterRoleDefinition{
				Rules: []rbacv1.PolicyRule{
					{
						APIGroups: []string{""},
						Resources: []string{"namespaces", "secrets", "serviceaccounts"},
						Verbs:     []string{"*"},
					},
					{
						APIGroups: []string{"rbac.authorization.k8s.io"},
						Resources: []string{"roles", "rolebindings"},
						Verbs:     []string{"*"},
					},
				},
			},
			expectedPermissions: map[string]storage.Access{
				string(resources.Namespace.GetResource()):      storage.Access_READ_WRITE_ACCESS,
				string(resources.Secret.GetResource()):         storage.Access_READ_WRITE_ACCESS,
				string(resources.ServiceAccount.GetResource()): storage.Access_READ_WRITE_ACCESS,
				string(resources.K8sRole.GetResource()):        storage.Access_READ_WRITE_ACCESS,
				string(resources.K8sRoleBinding.GetResource()): storage.Access_READ_WRITE_ACCESS,
			},
		},
		"unknown verbs grant no access": {
			clusterRoleDef: clusterviewv1alpha1.ClusterRoleDefinition{
				Rules: []rbacv1.PolicyRule{
					{
						APIGroups: []string{""},
						Resources: []string{"secrets"},
						Verbs:     []string{"unknown", "invalid"},
					},
				},
			},
			expectedPermissions: map[string]storage.Access{
				string(resources.Secret.GetResource()): storage.Access_NO_ACCESS,
			},
		},
		"empty rules": {
			clusterRoleDef: clusterviewv1alpha1.ClusterRoleDefinition{
				Rules: []rbacv1.PolicyRule{},
			},
			expectedPermissions: map[string]storage.Access{},
		},
		"rules with wrong API group are ignored": {
			clusterRoleDef: clusterviewv1alpha1.ClusterRoleDefinition{
				Rules: []rbacv1.PolicyRule{
					{
						APIGroups: []string{"apps"},
						Resources: []string{"secrets"}, // Right resource, wrong API group
						Verbs:     []string{"get"},
					},
				},
			},
			expectedPermissions: map[string]storage.Access{},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			permSet := ConvertClusterRoleToPermissionSet(tc.clusterRoleDef)

			// Verify PermissionSet structure
			require.NotNil(t, permSet)
			assert.NotEmpty(t, permSet.GetId(), "PermissionSet should have a generated ID")
			assert.NotNil(t, permSet.GetResourceToAccess())

			// Check minimum permission count if specified
			if tc.minPermissionCount > 0 {
				assert.GreaterOrEqual(t, len(permSet.GetResourceToAccess()), tc.minPermissionCount,
					"Expected at least %d permissions", tc.minPermissionCount)
			}

			// Verify expected permissions
			for resource, expectedAccess := range tc.expectedPermissions {
				actualAccess, exists := permSet.GetResourceToAccess()[resource]
				assert.True(t, exists, "Expected resource %q to be present", resource)
				assert.Equal(t, expectedAccess, actualAccess,
					"Expected %s access for resource %q, got %s",
					expectedAccess, resource, actualAccess)
			}

			// Verify unexpected resources are not present
			for _, resource := range tc.unexpectedResources {
				_, exists := permSet.GetResourceToAccess()[resource]
				assert.False(t, exists, "Did not expect resource %q to be present", resource)
			}

			// If we have exact expected permissions and no min count, verify exact match
			if len(tc.expectedPermissions) > 0 && tc.minPermissionCount == 0 {
				assert.Len(t, permSet.GetResourceToAccess(), len(tc.expectedPermissions),
					"Expected exactly %d permissions", len(tc.expectedPermissions))
			}
		})
	}
}

func TestComputeAccessLevel(t *testing.T) {
	tests := map[string]struct {
		verbs          []string
		expectedAccess storage.Access
	}{
		"read-only verbs": {
			verbs:          []string{"get", "list", "watch"},
			expectedAccess: storage.Access_READ_ACCESS,
		},
		"write verbs": {
			verbs:          []string{"create", "update", "delete"},
			expectedAccess: storage.Access_READ_WRITE_ACCESS,
		},
		"mixed read and write verbs": {
			verbs:          []string{"get", "list", "create"},
			expectedAccess: storage.Access_READ_WRITE_ACCESS,
		},
		"wildcard verb": {
			verbs:          []string{"*"},
			expectedAccess: storage.Access_READ_WRITE_ACCESS,
		},
		"wildcard with other verbs": {
			verbs:          []string{"get", "*", "list"},
			expectedAccess: storage.Access_READ_WRITE_ACCESS,
		},
		"patch verb": {
			verbs:          []string{"patch"},
			expectedAccess: storage.Access_READ_WRITE_ACCESS,
		},
		"deletecollection verb": {
			verbs:          []string{"deletecollection"},
			expectedAccess: storage.Access_READ_WRITE_ACCESS,
		},
		"unknown verbs": {
			verbs:          []string{"unknown", "invalid"},
			expectedAccess: storage.Access_NO_ACCESS,
		},
		"empty verbs": {
			verbs:          []string{},
			expectedAccess: storage.Access_NO_ACCESS,
		},
		"only get": {
			verbs:          []string{"get"},
			expectedAccess: storage.Access_READ_ACCESS,
		},
		"only create": {
			verbs:          []string{"create"},
			expectedAccess: storage.Access_READ_WRITE_ACCESS,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			access := computeAccessLevel(tc.verbs)
			assert.Equal(t, tc.expectedAccess, access,
				"Expected %s for verbs %v, got %s",
				tc.expectedAccess, tc.verbs, access)
		})
	}
}

func TestK8sToACSResourceMapping(t *testing.T) {
	expectedMappings := map[string]string{
		"namespaces":      string(resources.Namespace.GetResource()),
		"roles":           string(resources.K8sRole.GetResource()),
		"rolebindings":    string(resources.K8sRoleBinding.GetResource()),
		"secrets":         string(resources.Secret.GetResource()),
		"serviceaccounts": string(resources.ServiceAccount.GetResource()),
	}

	for k8sResource, expectedACSResource := range expectedMappings {
		t.Run(k8sResource, func(t *testing.T) {
			acsResource, exists := k8sToACSResourceMap[k8sResource]
			assert.True(t, exists, "Expected mapping for %q to exist", k8sResource)
			assert.Equal(t, expectedACSResource, acsResource,
				"Expected %q to map to %q, got %q",
				k8sResource, expectedACSResource, acsResource)
		})
	}
}

func TestConvertClusterRoleToPermissionSet_Integration(t *testing.T) {
	// This test simulates a realistic cluster-admin-like role
	clusterRoleDef := clusterviewv1alpha1.ClusterRoleDefinition{
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{"namespaces", "secrets", "serviceaccounts"},
				Verbs:     []string{"get", "list", "watch", "create", "update", "patch", "delete"},
			},
			{
				APIGroups: []string{"rbac.authorization.k8s.io"},
				Resources: []string{"roles", "rolebindings"},
				Verbs:     []string{"get", "list", "watch", "create", "update", "patch", "delete"},
			},
		},
	}

	permSet := ConvertClusterRoleToPermissionSet(clusterRoleDef)

	require.NotNil(t, permSet)
	assert.NotEmpty(t, permSet.GetId())
	assert.Len(t, permSet.GetResourceToAccess(), 5, "Should have all 5 base resources")

	// All resources should have READ_WRITE_ACCESS
	for _, resource := range []string{
		string(resources.Namespace.GetResource()),
		string(resources.Secret.GetResource()),
		string(resources.ServiceAccount.GetResource()),
		string(resources.K8sRole.GetResource()),
		string(resources.K8sRoleBinding.GetResource()),
	} {
		access, exists := permSet.GetResourceToAccess()[resource]
		assert.True(t, exists, "Expected resource %q to exist", resource)
		assert.Equal(t, storage.Access_READ_WRITE_ACCESS, access,
			"Expected READ_WRITE_ACCESS for %q", resource)
	}
}
