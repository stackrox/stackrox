package mapper

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

// mockACMClient is a test implementation of the ACM client interface for mapper testing
type mockACMClient struct {
	listFunc func(ctx context.Context, opts metav1.ListOptions) (*clusterviewv1alpha1.UserPermissionList, error)
}

func (m *mockACMClient) ListUserPermissions(ctx context.Context, opts metav1.ListOptions) (*clusterviewv1alpha1.UserPermissionList, error) {
	return m.listFunc(ctx, opts)
}

func (m *mockACMClient) GetUserPermission(ctx context.Context, name string, opts metav1.GetOptions) (*clusterviewv1alpha1.UserPermission, error) {
	return nil, errors.New("not implemented in mock")
}

func TestACMBasedMapper_FromUserDescriptor(t *testing.T) {
	tests := map[string]struct {
		mockListFunc      func(ctx context.Context, opts metav1.ListOptions) (*clusterviewv1alpha1.UserPermissionList, error)
		expectedRoleCount int
		expectedError     bool
		validateRoles     func(t *testing.T, roles []permissions.ResolvedRole)
	}{
		"successful retrieval with single role": {
			mockListFunc: func(ctx context.Context, opts metav1.ListOptions) (*clusterviewv1alpha1.UserPermissionList, error) {
				return &clusterviewv1alpha1.UserPermissionList{
					Items: []clusterviewv1alpha1.UserPermission{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "namespace-viewer",
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
										Cluster:    "prod-cluster",
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
				assert.Equal(t, "namespace-viewer", role.GetRoleName())
				assert.Equal(t, storage.Access_READ_ACCESS,
					role.GetPermissions()[string(resources.Namespace.GetResource())])
				assert.Contains(t, role.GetAccessScope().GetRules().GetIncludedClusters(), "prod-cluster")
			},
		},
		"multiple roles from ACM": {
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
										Cluster:    "cluster-a",
										Scope:      clusterviewv1alpha1.BindingScopeNamespace,
										Namespaces: []string{"team-a"},
									},
								},
							},
						},
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "rbac-admin",
							},
							Status: clusterviewv1alpha1.UserPermissionStatus{
								ClusterRoleDefinition: clusterviewv1alpha1.ClusterRoleDefinition{
									Rules: []rbacv1.PolicyRule{
										{
											APIGroups: []string{"rbac.authorization.k8s.io"},
											Resources: []string{"roles", "rolebindings"},
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

				var secretsReader, rbacAdmin permissions.ResolvedRole
				for _, role := range roles {
					if role.GetRoleName() == "secrets-reader" {
						secretsReader = role
					} else if role.GetRoleName() == "rbac-admin" {
						rbacAdmin = role
					}
				}

				require.NotNil(t, secretsReader)
				assert.Equal(t, storage.Access_READ_ACCESS,
					secretsReader.GetPermissions()[string(resources.Secret.GetResource())])
				assert.Len(t, secretsReader.GetAccessScope().GetRules().GetIncludedNamespaces(), 1)

				require.NotNil(t, rbacAdmin)
				assert.Equal(t, storage.Access_READ_WRITE_ACCESS,
					rbacAdmin.GetPermissions()[string(resources.K8sRole.GetResource())])
				assert.Contains(t, rbacAdmin.GetAccessScope().GetRules().GetIncludedClusters(), "cluster-b")
			},
		},
		"no base k8s resources filtered out": {
			mockListFunc: func(ctx context.Context, opts metav1.ListOptions) (*clusterviewv1alpha1.UserPermissionList, error) {
				return &clusterviewv1alpha1.UserPermissionList{
					Items: []clusterviewv1alpha1.UserPermission{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "deployment-admin",
							},
							Status: clusterviewv1alpha1.UserPermissionStatus{
								ClusterRoleDefinition: clusterviewv1alpha1.ClusterRoleDefinition{
									Rules: []rbacv1.PolicyRule{
										{
											APIGroups: []string{"apps"},
											Resources: []string{"deployments"},
											Verbs:     []string{"*"},
										},
									},
								},
								Bindings: []clusterviewv1alpha1.ClusterBinding{
									{
										Cluster:    "cluster-a",
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
		"empty list from ACM": {
			mockListFunc: func(ctx context.Context, opts metav1.ListOptions) (*clusterviewv1alpha1.UserPermissionList, error) {
				return &clusterviewv1alpha1.UserPermissionList{
					Items: []clusterviewv1alpha1.UserPermission{},
				}, nil
			},
			expectedRoleCount: 0,
			expectedError:     false,
		},
		"ACM client error": {
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

			mapper := NewACMBasedMapperWithClient(client)

			// UserDescriptor is not used by ACM mapper, so we can pass nil or empty
			userDescriptor := &permissions.UserDescriptor{
				UserID:     "test-user",
				Attributes: map[string][]string{},
			}

			roles, err := mapper.FromUserDescriptor(context.Background(), userDescriptor)

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

func TestNewACMBasedMapperWithClient(t *testing.T) {
	client := &mockACMClient{
		listFunc: func(ctx context.Context, opts metav1.ListOptions) (*clusterviewv1alpha1.UserPermissionList, error) {
			return &clusterviewv1alpha1.UserPermissionList{
				Items: []clusterviewv1alpha1.UserPermission{},
			}, nil
		},
	}

	mapper := NewACMBasedMapperWithClient(client)
	require.NotNil(t, mapper)

	// Verify it implements RoleMapper interface
	var _ permissions.RoleMapper = mapper
}
