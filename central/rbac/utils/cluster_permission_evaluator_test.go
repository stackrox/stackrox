package utils

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	roleMocks "github.com/stackrox/stackrox/central/rbac/k8srole/datastore/mocks"
	bindingMocks "github.com/stackrox/stackrox/central/rbac/k8srolebinding/datastore/mocks"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/search"
	"github.com/stretchr/testify/assert"
)

func TestIsClusterAdmin(t *testing.T) {
	cases := []struct {
		name          string
		inputRoles    []*storage.K8SRole
		inputBindings []*storage.K8SRoleBinding
		inputSubject  *storage.Subject
		expected      bool
	}{
		{
			name: "Cluster admin true",
			inputRoles: []*storage.K8SRole{
				{
					Id:          "role1",
					Name:        "cluster-admin",
					ClusterRole: true,
					ClusterId:   "cluster",
				},
			},
			inputBindings: []*storage.K8SRoleBinding{
				{
					RoleId: "role1",
					Subjects: []*storage.Subject{
						{
							Kind: storage.SubjectKind_SERVICE_ACCOUNT,
							Name: "foo",
						},
					},
					ClusterRole: true,
					ClusterId:   "cluster",
				},
			},
			inputSubject: &storage.Subject{
				Name: "foo",
				Kind: storage.SubjectKind_SERVICE_ACCOUNT,
			},
			expected: true,
		},
		{
			name: "Cluster admin false",
			inputRoles: []*storage.K8SRole{
				{
					Id:          "role1",
					Name:        "not-cluster-admin",
					ClusterId:   "cluster",
					ClusterRole: true,
					Rules: []*storage.PolicyRule{
						{
							Verbs: []string{
								"Get",
							},
							ApiGroups: []string{
								"custom",
							},
							Resources: []string{
								"pods",
							},
						},
					},
				},
			},
			inputBindings: []*storage.K8SRoleBinding{
				{
					RoleId:    "role1",
					ClusterId: "cluster",
					Subjects: []*storage.Subject{
						{
							Kind: storage.SubjectKind_SERVICE_ACCOUNT,
							Name: "foo",
						},
					},
					ClusterRole: true,
				},
			},
			inputSubject: &storage.Subject{
				Name: "foo",
				Kind: storage.SubjectKind_SERVICE_ACCOUNT,
			},
			expected: false,
		},
	}

	mockCtrl := gomock.NewController(t)

	clusterScopeQuery := search.NewQueryBuilder().
		AddExactMatches(search.ClusterID, "cluster").
		AddExactMatches(search.SubjectName, "foo").
		AddExactMatches(search.SubjectKind, storage.SubjectKind_SERVICE_ACCOUNT.String()).
		AddBools(search.ClusterRole, true).ProtoQuery()

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ctx := context.Background()
			mockBindingDatastore := bindingMocks.NewMockDataStore(mockCtrl)
			mockRoleDatastore := roleMocks.NewMockDataStore(mockCtrl)

			mockBindingDatastore.EXPECT().SearchRawRoleBindings(ctx, clusterScopeQuery).Return(c.inputBindings, nil).AnyTimes()
			mockRoleDatastore.EXPECT().GetRole(ctx, "role1").Return(c.inputRoles[0], true, nil).AnyTimes()

			evaluator := NewClusterPermissionEvaluator("cluster", mockRoleDatastore, mockBindingDatastore)
			assert.Equal(t, c.expected, evaluator.IsClusterAdmin(ctx, c.inputSubject))
		})
	}

}

func TestClusterPermissionsForSubject(t *testing.T) {
	cases := []struct {
		name          string
		inputRoles    []*storage.K8SRole
		inputBindings []*storage.K8SRoleBinding
		inputSubject  *storage.Subject
		expected      []*storage.PolicyRule
	}{

		{
			name: "get all pods",
			inputRoles: []*storage.K8SRole{
				{
					Id:          "role1",
					Name:        "get-pods-role",
					ClusterId:   "cluster",
					ClusterRole: true,
					Rules: []*storage.PolicyRule{
						{
							Verbs: []string{
								"get",
							},
							ApiGroups: []string{
								"",
							},
							Resources: []string{
								"pods",
							},
						},
					},
				},
			},
			inputBindings: []*storage.K8SRoleBinding{
				{
					RoleId:    "role1",
					ClusterId: "cluster",
					Subjects: []*storage.Subject{
						{
							Kind: storage.SubjectKind_SERVICE_ACCOUNT,
							Name: "foo",
						},
					},
					ClusterRole: true,
				},
				{
					RoleId:    "role1",
					ClusterId: "cluster",
					Subjects: []*storage.Subject{
						{
							Kind: storage.SubjectKind_SERVICE_ACCOUNT,
							Name: "foo",
						},
					},
					ClusterRole: true,
				},
			},
			inputSubject: &storage.Subject{
				Name: "foo",
				Kind: storage.SubjectKind_SERVICE_ACCOUNT,
			},
			expected: []*storage.PolicyRule{
				{
					Verbs:     []string{"get"},
					Resources: []string{"pods"},
					ApiGroups: []string{""},
				},
			},
		},
	}

	mockCtrl := gomock.NewController(t)
	mockBindingDatastore := bindingMocks.NewMockDataStore(mockCtrl)
	mockRoleDatastore := roleMocks.NewMockDataStore(mockCtrl)

	clusterScopeQuery := search.NewQueryBuilder().
		AddExactMatches(search.ClusterID, "cluster").
		AddExactMatches(search.SubjectName, "foo").
		AddExactMatches(search.SubjectKind, storage.SubjectKind_SERVICE_ACCOUNT.String()).
		AddBools(search.ClusterRole, true).ProtoQuery()

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ctx := context.Background()

			mockBindingDatastore.EXPECT().SearchRawRoleBindings(ctx, clusterScopeQuery).Return(c.inputBindings, nil).AnyTimes()
			mockRoleDatastore.EXPECT().GetRole(ctx, "role1").Return(c.inputRoles[0], true, nil).AnyTimes()

			evaluator := NewClusterPermissionEvaluator("cluster", mockRoleDatastore, mockBindingDatastore)
			assert.Equal(t, c.expected, evaluator.ForSubject(ctx, c.inputSubject).ToSlice())
		})
	}

}
