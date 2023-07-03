package utils

import (
	"context"
	"testing"

	roleDS "github.com/stackrox/rox/central/rbac/k8srole/datastore"
	roleMocks "github.com/stackrox/rox/central/rbac/k8srole/datastore/mocks"
	roleBindingDS "github.com/stackrox/rox/central/rbac/k8srolebinding/datastore"
	bindingMocks "github.com/stackrox/rox/central/rbac/k8srolebinding/datastore/mocks"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestNamespacePermissionsForSubject(t *testing.T) {
	cases := []struct {
		name          string
		inputRoles    []*storage.K8SRole
		inputBindings []*storage.K8SRoleBinding
		inputSubject  *storage.Subject
		expected      []*storage.PolicyRule
	}{

		{
			name: "get all pods and deployments",
			inputRoles: []*storage.K8SRole{
				{
					Id:        "role1",
					Name:      "get-pods-role",
					ClusterId: "cluster",
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
				{
					Id:        "role2",
					Name:      "get-deployments-role",
					ClusterId: "cluster",
					Rules: []*storage.PolicyRule{
						{
							Verbs: []string{
								"get",
							},
							ApiGroups: []string{
								"",
							},
							Resources: []string{
								"deployments",
							},
						},
					},
				},
			},
			inputBindings: []*storage.K8SRoleBinding{
				{
					Id:        "binding1",
					RoleId:    "role1",
					ClusterId: "cluster",
					Subjects: []*storage.Subject{
						{
							Kind:      storage.SubjectKind_SERVICE_ACCOUNT,
							Name:      "subject",
							Namespace: "namespace",
						},
					},
					ClusterRole: false,
					Namespace:   "namespace",
				},
				{
					Id:        "binding2",
					RoleId:    "role2",
					ClusterId: "cluster",
					Subjects: []*storage.Subject{
						{
							Kind:      storage.SubjectKind_SERVICE_ACCOUNT,
							Name:      "subject",
							Namespace: "namespace",
						},
					},
					ClusterRole: false,
					Namespace:   "namespace",
				},
			},
			inputSubject: &storage.Subject{
				Name:      "subject",
				Kind:      storage.SubjectKind_SERVICE_ACCOUNT,
				Namespace: "namespace",
			},
			expected: []*storage.PolicyRule{
				{
					Verbs:     []string{"get"},
					Resources: []string{"deployments", "pods"},
					ApiGroups: []string{""},
				},
			},
		},
	}

	mockCtrl := gomock.NewController(t)
	mockBindingDatastore := bindingMocks.NewMockDataStore(mockCtrl)
	mockRoleDatastore := roleMocks.NewMockDataStore(mockCtrl)

	namespaceScopeQuery := search.NewQueryBuilder().
		AddExactMatches(search.ClusterID, "cluster").
		AddExactMatches(search.Namespace, "namespace").
		AddExactMatches(search.SubjectName, "subject").
		AddExactMatches(search.SubjectKind, storage.SubjectKind_SERVICE_ACCOUNT.String()).ProtoQuery()

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ctx := sac.WithAllAccess(context.Background())

			mockBindingDatastore.EXPECT().SearchRawRoleBindings(ctx, namespaceScopeQuery).Return(c.inputBindings, nil).AnyTimes()
			mockRoleDatastore.EXPECT().GetRole(ctx, "role1").Return(c.inputRoles[0], true, nil).AnyTimes()
			mockRoleDatastore.EXPECT().GetRole(ctx, "role2").Return(c.inputRoles[1], true, nil).AnyTimes()

			evaluator := NewNamespacePermissionEvaluator("cluster", "namespace", mockRoleDatastore, mockBindingDatastore)
			assert.Equal(t, c.expected, evaluator.ForSubject(ctx, c.inputSubject).ToSlice())
		})
	}

}

func BenchmarkGetBindingsAndRoles(b *testing.B) {
	ctx := sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.K8sRole, resources.K8sRoleBinding, resources.K8sSubject)))

	pool := pgtest.ForT(b)

	roleStore := roleDS.GetTestPostgresDataStore(b, pool)
	bindingStore, err := roleBindingDS.GetTestPostgresDataStore(b, pool)
	require.NoError(b, err)

	bindings := fixtures.GetMultipleK8sRoleBindings(10000, 10)
	roles := fixtures.GetMultipleK8SRoles(10000)

	for _, role := range roles {
		require.NoError(b, roleStore.UpsertRole(ctx, role))
	}

	for _, binding := range bindings {
		require.NoError(b, bindingStore.UpsertRoleBinding(ctx, binding))
	}

	evaluator := NewNamespacePermissionEvaluator(bindings[500].GetClusterId(), bindings[500].GetNamespace(),
		roleStore, bindingStore).(*namespacePermissionEvaluator)

	subject := bindings[500].GetSubjects()[4]

	b.Run("run get bindings and roles", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			evaluator.getBindingsAndRoles(ctx, subject)
		}
	})
}
