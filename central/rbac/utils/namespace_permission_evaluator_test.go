//go:build sql_integration

package utils

import (
	"context"
	"testing"

	roleDS "github.com/stackrox/rox/central/rbac/k8srole/datastore"
	roleBindingDS "github.com/stackrox/rox/central/rbac/k8srolebinding/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNamespacePermissionsForSubject(t *testing.T) {
	ctx := sac.WithAllAccess(context.Background())

	clusterID := uuid.NewV4().String()

	testRoles := []*storage.K8SRole{
		{
			Id:        uuid.NewV4().String(),
			Name:      "get-pods-role",
			ClusterId: clusterID,
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
			Id:        uuid.NewV4().String(),
			Name:      "get-deployments-role",
			ClusterId: clusterID,
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
		{
			Id:        uuid.NewV4().String(),
			Name:      "get-replicasets-role",
			ClusterId: clusterID,
			Rules: []*storage.PolicyRule{
				{
					Verbs: []string{
						"get",
					},
					ApiGroups: []string{
						"",
					},
					Resources: []string{
						"replicasets",
					},
				},
			},
		},
		{
			Id:        uuid.NewV4().String(),
			Name:      "get-statefulsets-role",
			ClusterId: clusterID,
			Rules: []*storage.PolicyRule{
				{
					Verbs: []string{
						"get",
					},
					ApiGroups: []string{
						"",
					},
					Resources: []string{
						"statefulsets",
					},
				},
			},
		},
	}
	testBindings := []*storage.K8SRoleBinding{
		{
			Id:        uuid.NewV4().String(),
			RoleId:    testRoles[0].GetId(),
			ClusterId: clusterID,
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
			Id:        uuid.NewV4().String(),
			RoleId:    testRoles[1].GetId(),
			ClusterId: clusterID,
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
			Id:        uuid.NewV4().String(),
			RoleId:    testRoles[2].GetId(),
			ClusterId: clusterID,
			Subjects: []*storage.Subject{
				{
					Kind:      storage.SubjectKind_SERVICE_ACCOUNT,
					Name:      "subject",
					Namespace: "namespace",
				},
			},
			ClusterRole: true,
			Namespace:   "namespace",
		},
		{
			Id:        uuid.NewV4().String(),
			RoleId:    testRoles[3].GetId(),
			ClusterId: clusterID,
			Subjects: []*storage.Subject{
				{
					Kind:      storage.SubjectKind_SERVICE_ACCOUNT,
					Name:      "subject",
					Namespace: "namespace",
				},
			},
			ClusterRole: true,
		},
	}

	inputSubject := &storage.Subject{
		Name:      "subject",
		Kind:      storage.SubjectKind_SERVICE_ACCOUNT,
		Namespace: "namespace",
	}
	expectedResult := []*storage.PolicyRule{
		{
			Verbs:     []string{"get"},
			Resources: []string{"deployments", "pods", "replicasets"},
			ApiGroups: []string{""},
		},
	}

	pool := pgtest.ForT(t)
	roleStore := roleDS.GetTestPostgresDataStore(t, pool)
	bindingStore := roleBindingDS.GetTestPostgresDataStore(t, pool)
	for _, role := range testRoles {
		require.NoError(t, roleStore.UpsertRole(ctx, role))
	}

	for _, binding := range testBindings {
		require.NoError(t, bindingStore.UpsertRoleBinding(ctx, binding))
	}

	evaluator := NewNamespacePermissionEvaluator(clusterID, "namespace", roleStore, bindingStore)
	assert.Equal(t, expectedResult, evaluator.ForSubject(ctx, inputSubject).ToSlice())

	for _, role := range testRoles {
		require.NoError(t, roleStore.RemoveRole(ctx, role.GetId()))
	}
	for _, binding := range testBindings {
		require.NoError(t, bindingStore.RemoveRoleBinding(ctx, binding.GetId()))
	}
}

func BenchmarkGetBindingsAndRoles(b *testing.B) {
	ctx := sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedResourceLevelScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.K8sRole, resources.K8sRoleBinding, resources.K8sSubject)))

	pool := pgtest.ForT(b)

	roleStore := roleDS.GetTestPostgresDataStore(b, pool)
	bindingStore := roleBindingDS.GetTestPostgresDataStore(b, pool)

	roles := fixtures.GetMultipleK8SRoles(10000)
	bindings := fixtures.GetMultipleK8sRoleBindingsWithRole(10000, 10, roles)

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
