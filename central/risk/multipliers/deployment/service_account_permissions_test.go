package deployment

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	roleMocks "github.com/stackrox/rox/central/rbac/k8srole/datastore/mocks"
	bindingMocks "github.com/stackrox/rox/central/rbac/k8srolebinding/datastore/mocks"
	"github.com/stackrox/rox/central/risk/multipliers"
	saMocks "github.com/stackrox/rox/central/serviceaccount/datastore/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/assert"
)

func TestPermissionScore(t *testing.T) {
	deployment := multipliers.GetMockDeployment()
	clusterCases := []struct {
		name     string
		sa       *storage.ServiceAccount
		roles    []*storage.K8SRole
		bindings []*storage.K8SRoleBinding
		expected *storage.Risk_Result
	}{
		{
			name: "Test sa token not mounted",
			sa: &storage.ServiceAccount{
				Name:           "service-account",
				AutomountToken: false,
				ClusterId:      "cluster",
				Namespace:      "namespace",
			},
			roles:    []*storage.K8SRole{},
			bindings: []*storage.K8SRoleBinding{},
			expected: &storage.Risk_Result{
				Name: rbacConfigurationHeading,
				Factors: []*storage.Risk_Result_Factor{
					{Message: "Deployment is configured to automatically mount a token for service account \"service-account\""},
				},
				Score: 1.0,
			},
		},

		{
			name: "Test cluster admin",
			sa: &storage.ServiceAccount{
				Name:           "service-account",
				AutomountToken: true,
				ClusterId:      "cluster",
				Namespace:      "namespace",
			},
			roles: []*storage.K8SRole{
				{
					Id:          "role1",
					Name:        "effective admin",
					ClusterRole: true,
					Rules: []*storage.PolicyRule{
						{
							ApiGroups: []string{
								"",
							},
							Resources: []string{
								"*",
							},
							Verbs: []string{
								"*",
							},
						},
					},
				},
			},
			bindings: []*storage.K8SRoleBinding{
				{
					RoleId: "role1",
					Subjects: []*storage.Subject{
						{
							Kind:      storage.SubjectKind_SERVICE_ACCOUNT,
							Name:      "service-account",
							Namespace: "namespace",
						},
					},
					ClusterRole: true,
				},
			},
			expected: &storage.Risk_Result{
				Name: rbacConfigurationHeading,
				Factors: []*storage.Risk_Result_Factor{
					{Message: "Deployment is configured to automatically mount a token for service account \"service-account\""},
					{Message: "Service account \"service-account\" is configured to mount a token into the deployment by default"},
					{Message: "Service account \"service-account\" has been granted cluster admin privileges in the cluster"},
				},
				Score: 2.0,
			},
		},

		{
			name: "Test get on all resources in the cluster",
			sa: &storage.ServiceAccount{
				Name:           "service-account",
				AutomountToken: true,
				ClusterId:      "cluster",
				Namespace:      "namespace",
			},
			roles: []*storage.K8SRole{
				{
					Id:          "role1",
					Name:        "can get anything",
					ClusterRole: true,
					Rules: []*storage.PolicyRule{
						{
							ApiGroups: []string{
								"",
							},
							Resources: []string{
								"*",
							},
							Verbs: []string{
								"get",
							},
						},
					},
				},
			},
			bindings: []*storage.K8SRoleBinding{
				{
					RoleId: "role1",
					Subjects: []*storage.Subject{
						{
							Kind:      storage.SubjectKind_SERVICE_ACCOUNT,
							Name:      "service-account",
							Namespace: "namespace",
							ClusterId: deployment.GetClusterId(),
						},
					},
					ClusterRole: true,
				},
			},
			expected: &storage.Risk_Result{
				Name: rbacConfigurationHeading,
				Factors: []*storage.Risk_Result_Factor{
					{Message: "Deployment is configured to automatically mount a token for service account \"service-account\""},
					{Message: "Service account \"service-account\" is configured to mount a token into the deployment by default"},
					{Message: "Service account \"service-account\" has been granted the following permissions on resources in the cluster: get"},
				},
				Score: 1.0 + float32(138)/float32(maxPermissionScore),
			},
		},

		{
			name: "Test get on all resources, create on deployments in cluster",
			sa: &storage.ServiceAccount{
				Name:           "service-account",
				AutomountToken: true,
				ClusterId:      "cluster",
				Namespace:      "namespace",
			},
			roles: []*storage.K8SRole{
				{
					Id:          "role2",
					Name:        "can get anything, create deployments",
					ClusterRole: true,
					Rules: []*storage.PolicyRule{
						{
							ApiGroups: []string{
								"",
							},
							Resources: []string{
								"*",
							},
							Verbs: []string{
								"get",
							},
						},
						{
							ApiGroups: []string{
								"",
							},
							Resources: []string{
								"deployments",
							},
							Verbs: []string{
								"create",
							},
						},
					},
				},
			},
			bindings: []*storage.K8SRoleBinding{
				{
					RoleId: "role2",
					Subjects: []*storage.Subject{
						{
							Kind:      storage.SubjectKind_SERVICE_ACCOUNT,
							Name:      "service-account",
							Namespace: "namespace",
							ClusterId: "cluster",
						},
					},
					ClusterRole: true,
				},
			},
			expected: &storage.Risk_Result{
				Name: rbacConfigurationHeading,
				Factors: []*storage.Risk_Result_Factor{
					{Message: "Deployment is configured to automatically mount a token for service account \"service-account\""},
					{Message: "Service account \"service-account\" is configured to mount a token into the deployment by default"},
					{Message: "Service account \"service-account\" has been granted the following permissions on resources in the cluster: create, get"},
				},
				Score: 1.0 + float32(142)/float32(maxPermissionScore),
			},
		},
	}

	for _, c := range clusterCases {
		t.Run(c.name, func(t *testing.T) {
			ctx := context.Background()

			mockCtrl := gomock.NewController(t)

			mockSADatastore := saMocks.NewMockDataStore(mockCtrl)
			q := search.NewQueryBuilder().
				AddExactMatches(search.ClusterID, c.sa.ClusterId).
				AddExactMatches(search.Namespace, c.sa.Namespace).
				AddExactMatches(search.ServiceAccountName, c.sa.Name).ProtoQuery()
			mockSADatastore.EXPECT().SearchRawServiceAccounts(ctx, q).Return([]*storage.ServiceAccount{c.sa}, nil)

			mockBindingDatastore := bindingMocks.NewMockDataStore(mockCtrl)

			clusterScopeQuery := search.NewQueryBuilder().
				AddExactMatches(search.ClusterID, deployment.GetClusterId()).
				AddExactMatches(search.SubjectName, c.sa.Name).
				AddExactMatches(search.SubjectKind, storage.SubjectKind_SERVICE_ACCOUNT.String()).
				AddBools(search.ClusterRole, true).ProtoQuery()

			mockBindingDatastore.EXPECT().SearchRawRoleBindings(ctx, clusterScopeQuery).Return(c.bindings, nil).AnyTimes()

			namespaceScopeQuery := search.NewQueryBuilder().
				AddExactMatches(search.ClusterID, deployment.GetClusterId()).
				AddExactMatches(search.Namespace, deployment.GetNamespace()).
				AddExactMatches(search.SubjectName, c.sa.Name).
				AddExactMatches(search.SubjectKind, storage.SubjectKind_SERVICE_ACCOUNT.String()).
				AddBools(search.ClusterRole, false).ProtoQuery()

			mockBindingDatastore.EXPECT().SearchRawRoleBindings(ctx, namespaceScopeQuery).Return([]*storage.K8SRoleBinding{}, nil).AnyTimes()

			mockRoleDatastore := roleMocks.NewMockDataStore(mockCtrl)
			for i := range c.roles {
				mockRoleDatastore.EXPECT().GetRole(ctx, c.roles[i].GetId()).Return(c.roles[i], true, nil).AnyTimes()
			}

			mult := NewSAPermissionsMultiplier(mockRoleDatastore, mockBindingDatastore, mockSADatastore)
			result := mult.Score(ctx, deployment, nil)

			assert.Equal(t, c.expected, result)
			mockCtrl.Finish()
		})
	}

}
