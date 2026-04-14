package externalrolebroker

import (
	"context"
	"testing"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/sac/resources"
	clusterviewv1alpha1 "github.com/stolostron/cluster-lifecycle-api/clusterview/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// mockACMClient is a test implementation of the ACM client interface
type mockACMClient struct {
	listFunc func(ctx context.Context, opts metav1.ListOptions) (*clusterviewv1alpha1.UserPermissionList, error)
}

func (m *mockACMClient) ListUserPermissions(ctx context.Context, opts metav1.ListOptions) (*clusterviewv1alpha1.UserPermissionList, error) {
	return m.listFunc(ctx, opts)
}

func (m *mockACMClient) GetUserPermission(ctx context.Context, name string, opts metav1.GetOptions) (*clusterviewv1alpha1.UserPermission, error) {
	return nil, errors.New("not implemented in mock")
}

func TestGetResolvedRolesFromACM(t *testing.T) {
	tests := map[string]struct {
		mockListFunc      func(ctx context.Context, opts metav1.ListOptions) (*clusterviewv1alpha1.UserPermissionList, error)
		expectedRoleCount int
		expectedError     bool
		validateRoles     func(t *testing.T, roles []permissions.ResolvedRole)
	}{
		"empty list": {
			mockListFunc: func(ctx context.Context, opts metav1.ListOptions) (*clusterviewv1alpha1.UserPermissionList, error) {
				return &clusterviewv1alpha1.UserPermissionList{
					Items: []clusterviewv1alpha1.UserPermission{},
				}, nil
			},
			expectedRoleCount: 0,
			expectedError:     false,
		},
		"list with no base k8s resources": {
			mockListFunc: func(ctx context.Context, opts metav1.ListOptions) (*clusterviewv1alpha1.UserPermissionList, error) {
				return &clusterviewv1alpha1.UserPermissionList{
					Items: []clusterviewv1alpha1.UserPermission{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "non-base-permission",
							},
							Status: clusterviewv1alpha1.UserPermissionStatus{
								ClusterRoleDefinition: clusterviewv1alpha1.ClusterRoleDefinition{
									Rules: []rbacv1.PolicyRule{
										{
											APIGroups: []string{"apps"},
											Resources: []string{"deployments", "statefulsets"},
											Verbs:     []string{"get", "list"},
										},
									},
								},
								Bindings: []clusterviewv1alpha1.ClusterBinding{
									{
										Cluster:    "cluster-1",
										Scope:      clusterviewv1alpha1.BindingScopeCluster,
										Namespaces: []string{"*"},
									},
								},
							},
						},
					},
				}, nil
			},
			expectedRoleCount: 0,
			expectedError:     false,
		},
		"list with base k8s resources": {
			mockListFunc: func(ctx context.Context, opts metav1.ListOptions) (*clusterviewv1alpha1.UserPermissionList, error) {
				return &clusterviewv1alpha1.UserPermissionList{
					Items: []clusterviewv1alpha1.UserPermission{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "secrets-reader",
							},
							Status: clusterviewv1alpha1.UserPermissionStatus{
								ClusterRoleDefinition: clusterviewv1alpha1.ClusterRoleDefinition{
									Rules: []rbacv1.PolicyRule{
										{
											APIGroups: []string{""},
											Resources: []string{"secrets"},
											Verbs:     []string{"get", "list"},
										},
									},
								},
								Bindings: []clusterviewv1alpha1.ClusterBinding{
									{
										Cluster:    "cluster-1",
										Scope:      clusterviewv1alpha1.BindingScopeCluster,
										Namespaces: []string{"*"},
									},
								},
							},
						},
					},
				}, nil
			},
			expectedRoleCount: 1,
			expectedError:     false,
			validateRoles: func(t *testing.T, roles []permissions.ResolvedRole) {
				require.Len(t, roles, 1)
				role := roles[0]

				// Verify role name
				assert.Equal(t, "secrets-reader", role.GetRoleName())

				// Verify permissions
				permissions := role.GetPermissions()
				require.NotNil(t, permissions)
				assert.Equal(t, storage.Access_READ_ACCESS, permissions[string(resources.Secret.GetResource())])

				// Verify access scope
				accessScope := role.GetAccessScope()
				require.NotNil(t, accessScope)
				assert.Contains(t, accessScope.GetRules().GetIncludedClusters(), "cluster-1")
				assert.Empty(t, accessScope.GetRules().GetIncludedNamespaces())
			},
		},
		"multiple permissions with different access levels": {
			mockListFunc: func(ctx context.Context, opts metav1.ListOptions) (*clusterviewv1alpha1.UserPermissionList, error) {
				return &clusterviewv1alpha1.UserPermissionList{
					Items: []clusterviewv1alpha1.UserPermission{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "namespace-reader",
							},
							Status: clusterviewv1alpha1.UserPermissionStatus{
								ClusterRoleDefinition: clusterviewv1alpha1.ClusterRoleDefinition{
									Rules: []rbacv1.PolicyRule{
										{
											APIGroups: []string{""},
											Resources: []string{"namespaces"},
											Verbs:     []string{"get", "list"},
										},
									},
								},
								Bindings: []clusterviewv1alpha1.ClusterBinding{
									{
										Cluster:    "cluster-a",
										Scope:      clusterviewv1alpha1.BindingScopeNamespace,
										Namespaces: []string{"default", "kube-system"},
									},
								},
							},
						},
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "secrets-admin",
							},
							Status: clusterviewv1alpha1.UserPermissionStatus{
								ClusterRoleDefinition: clusterviewv1alpha1.ClusterRoleDefinition{
									Rules: []rbacv1.PolicyRule{
										{
											APIGroups: []string{""},
											Resources: []string{"secrets"},
											Verbs:     []string{"*"},
										},
									},
								},
								Bindings: []clusterviewv1alpha1.ClusterBinding{
									{
										Cluster:    "cluster-b",
										Scope:      clusterviewv1alpha1.BindingScopeCluster,
										Namespaces: []string{"*"},
									},
								},
							},
						},
					},
				}, nil
			},
			expectedRoleCount: 2,
			expectedError:     false,
			validateRoles: func(t *testing.T, roles []permissions.ResolvedRole) {
				require.Len(t, roles, 2)

				// Verify first role (namespace-reader)
				var namespaceReader, secretsAdmin permissions.ResolvedRole
				for _, role := range roles {
					if role.GetRoleName() == "namespace-reader" {
						namespaceReader = role
					} else if role.GetRoleName() == "secrets-admin" {
						secretsAdmin = role
					}
				}

				require.NotNil(t, namespaceReader)
				assert.Equal(t, storage.Access_READ_ACCESS,
					namespaceReader.GetPermissions()[string(resources.Namespace.GetResource())])
				assert.Len(t, namespaceReader.GetAccessScope().GetRules().GetIncludedNamespaces(), 2)

				require.NotNil(t, secretsAdmin)
				assert.Equal(t, storage.Access_READ_WRITE_ACCESS,
					secretsAdmin.GetPermissions()[string(resources.Secret.GetResource())])
				assert.Contains(t, secretsAdmin.GetAccessScope().GetRules().GetIncludedClusters(), "cluster-b")
			},
		},
		"acm client returns error": {
			mockListFunc: func(ctx context.Context, opts metav1.ListOptions) (*clusterviewv1alpha1.UserPermissionList, error) {
				return nil, errors.New("ACM API unavailable")
			},
			expectedRoleCount: 0,
			expectedError:     true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			client := &mockACMClient{
				listFunc: tc.mockListFunc,
			}

			roles, err := GetResolvedRolesFromACM(context.Background(), client)

			if tc.expectedError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Len(t, roles, tc.expectedRoleCount)

			if tc.validateRoles != nil {
				tc.validateRoles(t, roles)
			}
		})
	}
}

func TestGetResolvedRolesFromACM_IntegrationExample(t *testing.T) {
	// This test demonstrates the full integration of all conversion functions
	client := &mockACMClient{
		listFunc: func(ctx context.Context, opts metav1.ListOptions) (*clusterviewv1alpha1.UserPermissionList, error) {
			return &clusterviewv1alpha1.UserPermissionList{
				Items: []clusterviewv1alpha1.UserPermission{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "rbac-admin",
						},
						Status: clusterviewv1alpha1.UserPermissionStatus{
							ClusterRoleDefinition: clusterviewv1alpha1.ClusterRoleDefinition{
								Rules: []rbacv1.PolicyRule{
									{
										APIGroups: []string{""},
										Resources: []string{"namespaces", "secrets", "serviceaccounts"},
										Verbs:     []string{"get", "list", "create", "update"},
									},
									{
										APIGroups: []string{"rbac.authorization.k8s.io"},
										Resources: []string{"roles", "rolebindings"},
										Verbs:     []string{"*"},
									},
								},
							},
							Bindings: []clusterviewv1alpha1.ClusterBinding{
								{
									Cluster:    "prod-cluster",
									Scope:      clusterviewv1alpha1.BindingScopeCluster,
									Namespaces: []string{"*"},
								},
								{
									Cluster:    "dev-cluster",
									Scope:      clusterviewv1alpha1.BindingScopeNamespace,
									Namespaces: []string{"team-a", "team-b"},
								},
							},
						},
					},
				},
			}, nil
		},
	}

	roles, err := GetResolvedRolesFromACM(context.Background(), client)

	require.NoError(t, err)
	require.Len(t, roles, 1)

	role := roles[0]
	assert.Equal(t, "rbac-admin", role.GetRoleName())

	// Verify all 5 base resources have permissions
	perms := role.GetPermissions()
	assert.Len(t, perms, 5)
	assert.Equal(t, storage.Access_READ_WRITE_ACCESS, perms[string(resources.Namespace.GetResource())])
	assert.Equal(t, storage.Access_READ_WRITE_ACCESS, perms[string(resources.Secret.GetResource())])
	assert.Equal(t, storage.Access_READ_WRITE_ACCESS, perms[string(resources.ServiceAccount.GetResource())])
	assert.Equal(t, storage.Access_READ_WRITE_ACCESS, perms[string(resources.K8sRole.GetResource())])
	assert.Equal(t, storage.Access_READ_WRITE_ACCESS, perms[string(resources.K8sRoleBinding.GetResource())])

	// Verify access scope includes cluster and namespaces
	scope := role.GetAccessScope()
	require.NotNil(t, scope)
	assert.Contains(t, scope.GetRules().GetIncludedClusters(), "prod-cluster")
	assert.Len(t, scope.GetRules().GetIncludedNamespaces(), 2)
}
