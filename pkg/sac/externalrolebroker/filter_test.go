package externalrolebroker

import (
	"testing"

	clusterviewv1alpha1 "github.com/stolostron/cluster-lifecycle-api/clusterview/v1alpha1"
	"github.com/stretchr/testify/assert"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestFilterUserPermissionsForBaseK8sResources(t *testing.T) {
	tests := map[string]struct {
		permissions []clusterviewv1alpha1.UserPermission
		expected    int
	}{
		"empty list": {
			permissions: []clusterviewv1alpha1.UserPermission{},
			expected:    0,
		},
		"permission with namespace access": {
			permissions: []clusterviewv1alpha1.UserPermission{
				makeUserPermission("test-1", []rbacv1.PolicyRule{
					{
						APIGroups: []string{""},
						Resources: []string{"namespaces"},
						Verbs:     []string{"get", "list"},
					},
				}),
			},
			expected: 1,
		},
		"permission with secret access": {
			permissions: []clusterviewv1alpha1.UserPermission{
				makeUserPermission("test-2", []rbacv1.PolicyRule{
					{
						APIGroups: []string{""},
						Resources: []string{"secrets"},
						Verbs:     []string{"get", "list", "create"},
					},
				}),
			},
			expected: 1,
		},
		"permission with role and rolebinding access": {
			permissions: []clusterviewv1alpha1.UserPermission{
				makeUserPermission("test-3", []rbacv1.PolicyRule{
					{
						APIGroups: []string{"rbac.authorization.k8s.io"},
						Resources: []string{"roles", "rolebindings"},
						Verbs:     []string{"get", "list", "create", "update", "delete"},
					},
				}),
			},
			expected: 1,
		},
		"permission with clusterrole and clusterrolebinding access": {
			permissions: []clusterviewv1alpha1.UserPermission{
				makeUserPermission("test-3b", []rbacv1.PolicyRule{
					{
						APIGroups: []string{"rbac.authorization.k8s.io"},
						Resources: []string{"clusterroles", "clusterrolebindings"},
						Verbs:     []string{"get", "list", "create", "update", "delete"},
					},
				}),
			},
			expected: 1,
		},
		"permission with serviceaccount access": {
			permissions: []clusterviewv1alpha1.UserPermission{
				makeUserPermission("test-4", []rbacv1.PolicyRule{
					{
						APIGroups: []string{""},
						Resources: []string{"serviceaccounts"},
						Verbs:     []string{"get", "list"},
					},
				}),
			},
			expected: 1,
		},
		"permission with wildcard resources in core group": {
			permissions: []clusterviewv1alpha1.UserPermission{
				makeUserPermission("test-5", []rbacv1.PolicyRule{
					{
						APIGroups: []string{""},
						Resources: []string{"*"},
						Verbs:     []string{"*"},
					},
				}),
			},
			expected: 1,
		},
		"permission with wildcard apiGroups": {
			permissions: []clusterviewv1alpha1.UserPermission{
				makeUserPermission("test-6", []rbacv1.PolicyRule{
					{
						APIGroups: []string{"*"},
						Resources: []string{"secrets"},
						Verbs:     []string{"get"},
					},
				}),
			},
			expected: 1,
		},
		"permission with non-base resources only": {
			permissions: []clusterviewv1alpha1.UserPermission{
				makeUserPermission("test-7", []rbacv1.PolicyRule{
					{
						APIGroups: []string{"apps"},
						Resources: []string{"deployments", "statefulsets"},
						Verbs:     []string{"get", "list"},
					},
				}),
			},
			expected: 0,
		},
		"permission with pods only (not a base resource)": {
			permissions: []clusterviewv1alpha1.UserPermission{
				makeUserPermission("test-8", []rbacv1.PolicyRule{
					{
						APIGroups: []string{""},
						Resources: []string{"pods"},
						Verbs:     []string{"get", "list"},
					},
				}),
			},
			expected: 0,
		},
		"mixed permissions - some match, some don't": {
			permissions: []clusterviewv1alpha1.UserPermission{
				makeUserPermission("match-1", []rbacv1.PolicyRule{
					{
						APIGroups: []string{""},
						Resources: []string{"secrets"},
						Verbs:     []string{"get"},
					},
				}),
				makeUserPermission("no-match-1", []rbacv1.PolicyRule{
					{
						APIGroups: []string{"apps"},
						Resources: []string{"deployments"},
						Verbs:     []string{"get"},
					},
				}),
				makeUserPermission("match-2", []rbacv1.PolicyRule{
					{
						APIGroups: []string{"rbac.authorization.k8s.io"},
						Resources: []string{"roles"},
						Verbs:     []string{"create"},
					},
				}),
				makeUserPermission("no-match-2", []rbacv1.PolicyRule{
					{
						APIGroups: []string{""},
						Resources: []string{"configmaps"},
						Verbs:     []string{"get"},
					},
				}),
			},
			expected: 2,
		},
		"permission with multiple rules - one matches": {
			permissions: []clusterviewv1alpha1.UserPermission{
				makeUserPermission("multi-rule", []rbacv1.PolicyRule{
					{
						APIGroups: []string{"apps"},
						Resources: []string{"deployments"},
						Verbs:     []string{"get"},
					},
					{
						APIGroups: []string{""},
						Resources: []string{"secrets"}, // This one matches
						Verbs:     []string{"get"},
					},
				}),
			},
			expected: 1,
		},
		"permission with subresources": {
			permissions: []clusterviewv1alpha1.UserPermission{
				makeUserPermission("subresource", []rbacv1.PolicyRule{
					{
						APIGroups: []string{""},
						Resources: []string{"secrets/status"}, // Should still match "secrets"
						Verbs:     []string{"get"},
					},
				}),
			},
			expected: 1,
		},
		"permission with empty APIGroups": {
			permissions: []clusterviewv1alpha1.UserPermission{
				makeUserPermission("empty-apigroups", []rbacv1.PolicyRule{
					{
						APIGroups: []string{},
						Resources: []string{"secrets"},
						Verbs:     []string{"get"},
					},
				}),
			},
			expected: 0,
		},
		"permission with all base resources": {
			permissions: []clusterviewv1alpha1.UserPermission{
				makeUserPermission("all-base", []rbacv1.PolicyRule{
					{
						APIGroups: []string{""},
						Resources: []string{"namespaces", "secrets", "serviceaccounts"},
						Verbs:     []string{"get", "list"},
					},
					{
						APIGroups: []string{"rbac.authorization.k8s.io"},
						Resources: []string{"roles", "rolebindings"},
						Verbs:     []string{"get", "list"},
					},
				}),
			},
			expected: 1,
		},
		"permission with StackRox API alert access": {
			permissions: []clusterviewv1alpha1.UserPermission{
				makeUserPermission("stackrox-alert", []rbacv1.PolicyRule{
					{
						APIGroups: []string{"api.stackrox.io"},
						Resources: []string{"alerts"},
						Verbs:     []string{"get", "list"},
					},
				}),
			},
			expected: 1,
		},
		"permission with StackRox API deployment access": {
			permissions: []clusterviewv1alpha1.UserPermission{
				makeUserPermission("stackrox-deployment", []rbacv1.PolicyRule{
					{
						APIGroups: []string{"api.stackrox.io"},
						Resources: []string{"deployments"},
						Verbs:     []string{"get", "list", "create"},
					},
				}),
			},
			expected: 1,
		},
		"permission with multiple StackRox API resources": {
			permissions: []clusterviewv1alpha1.UserPermission{
				makeUserPermission("stackrox-multi", []rbacv1.PolicyRule{
					{
						APIGroups: []string{"api.stackrox.io"},
						Resources: []string{"compliances", "detections", "networkgraphs"},
						Verbs:     []string{"get", "list"},
					},
				}),
			},
			expected: 1,
		},
		"permission with vulnerability management resources": {
			permissions: []clusterviewv1alpha1.UserPermission{
				makeUserPermission("stackrox-vuln", []rbacv1.PolicyRule{
					{
						APIGroups: []string{"api.stackrox.io"},
						Resources: []string{"vulnerabilitymanagementrequests", "vulnerabilitymanagementapprovals"},
						Verbs:     []string{"get", "list", "create"},
					},
				}),
			},
			expected: 1,
		},
		"permission with StackRox API wrong API group": {
			permissions: []clusterviewv1alpha1.UserPermission{
				makeUserPermission("wrong-apigroup", []rbacv1.PolicyRule{
					{
						APIGroups: []string{"apps"},
						Resources: []string{"alerts"},
						Verbs:     []string{"get"},
					},
				}),
			},
			expected: 0,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			result := FilterUserPermissionsForBaseK8sResources(tc.permissions)
			assert.Len(t, result, tc.expected, "Expected %d permissions to be filtered, got %d", tc.expected, len(result))
		})
	}
}

func TestHasBaseK8sResources(t *testing.T) {
	tests := map[string]struct {
		permission *clusterviewv1alpha1.UserPermission
		expected   bool
	}{
		"has namespace resource": {
			permission: &clusterviewv1alpha1.UserPermission{
				Status: clusterviewv1alpha1.UserPermissionStatus{
					ClusterRoleDefinition: clusterviewv1alpha1.ClusterRoleDefinition{
						Rules: []rbacv1.PolicyRule{
							{
								APIGroups: []string{""},
								Resources: []string{"namespaces"},
								Verbs:     []string{"get"},
							},
						},
					},
				},
			},
			expected: true,
		},
		"has role resource": {
			permission: &clusterviewv1alpha1.UserPermission{
				Status: clusterviewv1alpha1.UserPermissionStatus{
					ClusterRoleDefinition: clusterviewv1alpha1.ClusterRoleDefinition{
						Rules: []rbacv1.PolicyRule{
							{
								APIGroups: []string{"rbac.authorization.k8s.io"},
								Resources: []string{"roles"},
								Verbs:     []string{"get"},
							},
						},
					},
				},
			},
			expected: true,
		},
		"no base resources": {
			permission: &clusterviewv1alpha1.UserPermission{
				Status: clusterviewv1alpha1.UserPermissionStatus{
					ClusterRoleDefinition: clusterviewv1alpha1.ClusterRoleDefinition{
						Rules: []rbacv1.PolicyRule{
							{
								APIGroups: []string{"apps"},
								Resources: []string{"deployments"},
								Verbs:     []string{"get"},
							},
						},
					},
				},
			},
			expected: false,
		},
		"empty rules": {
			permission: &clusterviewv1alpha1.UserPermission{
				Status: clusterviewv1alpha1.UserPermissionStatus{
					ClusterRoleDefinition: clusterviewv1alpha1.ClusterRoleDefinition{
						Rules: []rbacv1.PolicyRule{},
					},
				},
			},
			expected: false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			result := hasBaseK8sResources(tc.permission)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestHasConfiguredResource(t *testing.T) {
	tests := map[string]struct {
		rule     rbacv1.PolicyRule
		expected bool
	}{
		"core API group with namespace": {
			rule: rbacv1.PolicyRule{
				APIGroups: []string{""},
				Resources: []string{"namespaces"},
			},
			expected: true,
		},
		"core API group with secret": {
			rule: rbacv1.PolicyRule{
				APIGroups: []string{""},
				Resources: []string{"secrets"},
			},
			expected: true,
		},
		"core API group with serviceaccount": {
			rule: rbacv1.PolicyRule{
				APIGroups: []string{""},
				Resources: []string{"serviceaccounts"},
			},
			expected: true,
		},
		"rbac API group with role": {
			rule: rbacv1.PolicyRule{
				APIGroups: []string{"rbac.authorization.k8s.io"},
				Resources: []string{"roles"},
			},
			expected: true,
		},
		"rbac API group with rolebinding": {
			rule: rbacv1.PolicyRule{
				APIGroups: []string{"rbac.authorization.k8s.io"},
				Resources: []string{"rolebindings"},
			},
			expected: true,
		},
		"rbac API group with clusterrole": {
			rule: rbacv1.PolicyRule{
				APIGroups: []string{"rbac.authorization.k8s.io"},
				Resources: []string{"clusterroles"},
			},
			expected: true,
		},
		"rbac API group with clusterrolebinding": {
			rule: rbacv1.PolicyRule{
				APIGroups: []string{"rbac.authorization.k8s.io"},
				Resources: []string{"clusterrolebindings"},
			},
			expected: true,
		},
		"wildcard API group with configured resource": {
			rule: rbacv1.PolicyRule{
				APIGroups: []string{"*"},
				Resources: []string{"secrets"},
			},
			expected: true,
		},
		"wildcard resource with core API group": {
			rule: rbacv1.PolicyRule{
				APIGroups: []string{""},
				Resources: []string{"*"},
			},
			expected: true,
		},
		"wildcard resource with rbac API group": {
			rule: rbacv1.PolicyRule{
				APIGroups: []string{"rbac.authorization.k8s.io"},
				Resources: []string{"*"},
			},
			expected: true,
		},
		"wildcard resource and API group": {
			rule: rbacv1.PolicyRule{
				APIGroups: []string{"*"},
				Resources: []string{"*"},
			},
			expected: true,
		},
		"non-configured resource": {
			rule: rbacv1.PolicyRule{
				APIGroups: []string{""},
				Resources: []string{"pods"},
			},
			expected: false,
		},
		"wrong API group for resource": {
			rule: rbacv1.PolicyRule{
				APIGroups: []string{"apps"},
				Resources: []string{"secrets"},
			},
			expected: false,
		},
		"rbac resource with wrong API group": {
			rule: rbacv1.PolicyRule{
				APIGroups: []string{""},
				Resources: []string{"roles"},
			},
			expected: false,
		},
		"core resource with wrong API group": {
			rule: rbacv1.PolicyRule{
				APIGroups: []string{"rbac.authorization.k8s.io"},
				Resources: []string{"secrets"},
			},
			expected: false,
		},
		"empty API groups": {
			rule: rbacv1.PolicyRule{
				APIGroups: []string{},
				Resources: []string{"secrets"},
			},
			expected: false,
		},
		"subresource for secrets with correct API group": {
			rule: rbacv1.PolicyRule{
				APIGroups: []string{""},
				Resources: []string{"secrets/status"},
			},
			expected: true,
		},
		"multiple API groups with one matching": {
			rule: rbacv1.PolicyRule{
				APIGroups: []string{"apps", ""},
				Resources: []string{"secrets"},
			},
			expected: true,
		},
		"multiple resources with one configured": {
			rule: rbacv1.PolicyRule{
				APIGroups: []string{""},
				Resources: []string{"pods", "secrets", "configmaps"},
			},
			expected: true,
		},
		"empty resources": {
			rule: rbacv1.PolicyRule{
				APIGroups: []string{""},
				Resources: []string{},
			},
			expected: false,
		},
		"StackRox API group with alert": {
			rule: rbacv1.PolicyRule{
				APIGroups: []string{"api.stackrox.io"},
				Resources: []string{"alerts"},
			},
			expected: true,
		},
		"StackRox API group with deployment": {
			rule: rbacv1.PolicyRule{
				APIGroups: []string{"api.stackrox.io"},
				Resources: []string{"deployments"},
			},
			expected: true,
		},
		"StackRox API group with compliance": {
			rule: rbacv1.PolicyRule{
				APIGroups: []string{"api.stackrox.io"},
				Resources: []string{"compliances"},
			},
			expected: true,
		},
		"StackRox API group with detection": {
			rule: rbacv1.PolicyRule{
				APIGroups: []string{"api.stackrox.io"},
				Resources: []string{"detections"},
			},
			expected: true,
		},
		"StackRox API group with networkgraph": {
			rule: rbacv1.PolicyRule{
				APIGroups: []string{"api.stackrox.io"},
				Resources: []string{"networkgraphs"},
			},
			expected: true,
		},
		"StackRox API group with vulnerabilitymanagementrequest": {
			rule: rbacv1.PolicyRule{
				APIGroups: []string{"api.stackrox.io"},
				Resources: []string{"vulnerabilitymanagementrequests"},
			},
			expected: true,
		},
		"StackRox API group with vulnerabilitymanagementapproval": {
			rule: rbacv1.PolicyRule{
				APIGroups: []string{"api.stackrox.io"},
				Resources: []string{"vulnerabilitymanagementapprovals"},
			},
			expected: true,
		},
		"StackRox API resources with wrong API group": {
			rule: rbacv1.PolicyRule{
				APIGroups: []string{"apps"},
				Resources: []string{"alerts"},
			},
			expected: false,
		},
		"wildcard resource with StackRox API group": {
			rule: rbacv1.PolicyRule{
				APIGroups: []string{"api.stackrox.io"},
				Resources: []string{"*"},
			},
			expected: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			result := hasConfiguredResource(tc.rule)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// Helper function to create a UserPermission with given rules
func makeUserPermission(name string, rules []rbacv1.PolicyRule) clusterviewv1alpha1.UserPermission {
	return clusterviewv1alpha1.UserPermission{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Status: clusterviewv1alpha1.UserPermissionStatus{
			ClusterRoleDefinition: clusterviewv1alpha1.ClusterRoleDefinition{
				Rules: rules,
			},
		},
	}
}
