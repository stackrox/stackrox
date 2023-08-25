//go:build sql_integration

package utils

import (
	"context"
	"testing"

	roleDS "github.com/stackrox/rox/central/rbac/k8srole/datastore"
	roleBindingDS "github.com/stackrox/rox/central/rbac/k8srolebinding/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsClusterAdmin(t *testing.T) {
	clusterID := uuid.NewV4().String()
	role1ID := uuid.NewV4().String()

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
					Id:          role1ID,
					Name:        "cluster-admin",
					ClusterRole: true,
					ClusterId:   clusterID,
				},
			},
			inputBindings: []*storage.K8SRoleBinding{
				{
					RoleId: role1ID,
					Subjects: []*storage.Subject{
						{
							Kind: storage.SubjectKind_SERVICE_ACCOUNT,
							Name: "foo",
						},
					},
					ClusterRole: true,
					ClusterId:   clusterID,
					Id:          uuid.NewV4().String(),
				},
				{
					RoleId: role1ID,
					Subjects: []*storage.Subject{
						{
							Kind: storage.SubjectKind_SERVICE_ACCOUNT,
							Name: "foo",
						},
					},
					ClusterRole: true,
					ClusterId:   clusterID,
					Namespace:   "something",
					Id:          uuid.NewV4().String(),
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
					Id:          role1ID,
					Name:        "not-cluster-admin",
					ClusterId:   clusterID,
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
					RoleId:    role1ID,
					Id:        uuid.NewV4().String(),
					ClusterId: clusterID,
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

	pool := pgtest.ForT(t)
	roleStore := roleDS.GetTestPostgresDataStore(t, pool)
	bindingStore := roleBindingDS.GetTestPostgresDataStore(t, pool)

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ctx := sac.WithAllAccess(context.Background())
			for _, role := range c.inputRoles {
				require.NoError(t, roleStore.UpsertRole(ctx, role))
			}
			for _, binding := range c.inputBindings {
				require.NoError(t, bindingStore.UpsertRoleBinding(ctx, binding))
			}
			evaluator := NewClusterPermissionEvaluator(clusterID, roleStore, bindingStore)
			assert.Equal(t, c.expected, evaluator.IsClusterAdmin(ctx, c.inputSubject))

			for _, role := range c.inputRoles {
				require.NoError(t, roleStore.RemoveRole(ctx, role.GetId()))
			}
			for _, binding := range c.inputBindings {
				require.NoError(t, bindingStore.RemoveRoleBinding(ctx, binding.GetId()))
			}
		})
	}

}

func TestClusterPermissionsForSubject(t *testing.T) {
	clusterID := uuid.NewV4().String()
	role1ID := uuid.NewV4().String()

	inputRoles := []*storage.K8SRole{
		{
			Id:          role1ID,
			Name:        "get-pods-role",
			ClusterId:   clusterID,
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
	}
	inputBindings := []*storage.K8SRoleBinding{
		{
			Id:        uuid.NewV4().String(),
			RoleId:    role1ID,
			ClusterId: clusterID,
			Subjects: []*storage.Subject{
				{
					Kind: storage.SubjectKind_SERVICE_ACCOUNT,
					Name: "foo",
				},
			},
			ClusterRole: true,
		},
		{
			Id:        uuid.NewV4().String(),
			RoleId:    role1ID,
			ClusterId: clusterID,
			Subjects: []*storage.Subject{
				{
					Kind: storage.SubjectKind_SERVICE_ACCOUNT,
					Name: "foo",
				},
			},
			ClusterRole: true,
		},
	}
	inputSubject := &storage.Subject{
		Name: "foo",
		Kind: storage.SubjectKind_SERVICE_ACCOUNT,
	}
	expected := []*storage.PolicyRule{
		{
			Verbs:     []string{"get"},
			Resources: []string{"pods"},
			ApiGroups: []string{""},
		},
	}

	pool := pgtest.ForT(t)
	roleStore := roleDS.GetTestPostgresDataStore(t, pool)
	bindingStore := roleBindingDS.GetTestPostgresDataStore(t, pool)

	ctx := sac.WithAllAccess(context.Background())

	for _, role := range inputRoles {
		require.NoError(t, roleStore.UpsertRole(ctx, role))
	}

	for _, binding := range inputBindings {
		require.NoError(t, bindingStore.UpsertRoleBinding(ctx, binding))
	}

	evaluator := NewClusterPermissionEvaluator(clusterID, roleStore, bindingStore)
	assert.Equal(t, expected, evaluator.ForSubject(ctx, inputSubject).ToSlice())

	for _, role := range inputRoles {
		require.NoError(t, roleStore.RemoveRole(ctx, role.GetId()))
	}
	for _, binding := range inputBindings {
		require.NoError(t, bindingStore.RemoveRoleBinding(ctx, binding.GetId()))
	}
}
