package rbac

import (
	"fmt"
	"testing"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func TestStore(t *testing.T) {
	// Namespace: n1
	// Role: r1
	// Bindings:
	//  - b1 -> r1
	//  - b2 -> r1
	// Cluster role: r2
	// Cluster bindings:
	//  - b3 -> r2
	//  - b4 -> r2
	roles := []*v1.Role{
		{
			ObjectMeta: metav1.ObjectMeta{
				UID:       types.UID("r1"),
				Name:      "r1",
				Namespace: "n1",
			},
		},
	}
	clusterRoles := []*v1.ClusterRole{
		{
			ObjectMeta: metav1.ObjectMeta{
				UID:       types.UID("r2"),
				Name:      "r2",
				Namespace: "n1",
			},
		},
	}
	bindings := []*v1.RoleBinding{
		{
			ObjectMeta: metav1.ObjectMeta{
				UID:       types.UID("b1"),
				Name:      "b1",
				Namespace: "n1",
			},
			RoleRef: v1.RoleRef{
				Name:     "r1",
				Kind:     "Role",
				APIGroup: "rbac.authorization.k8s.io",
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				UID:       types.UID("b2"),
				Name:      "b2",
				Namespace: "n1",
			},
			RoleRef: v1.RoleRef{
				Name:     "r1",
				Kind:     "Role",
				APIGroup: "rbac.authorization.k8s.io",
			},
		},
	}
	clusterBindings := []*v1.ClusterRoleBinding{
		{
			ObjectMeta: metav1.ObjectMeta{
				UID:       types.UID("b3"),
				Name:      "b3",
				Namespace: "n1",
			},
			RoleRef: v1.RoleRef{
				Name:     "r2",
				Kind:     "ClusterRole",
				APIGroup: "rbac.authorization.k8s.io",
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				UID:       types.UID("b4"),
				Name:      "b4",
				Namespace: "n1",
			},
			RoleRef: v1.RoleRef{
				Name:     "r2",
				Kind:     "ClusterRole",
				APIGroup: "rbac.authorization.k8s.io",
			},
		},
	}

	tested := NewStore().(*storeImpl)
	dispatcher := NewDispatcher(tested)

	// Add a binding with no role, should get a binding update with no role id.
	event := dispatcher.ProcessEvent(bindings[0], nil, central.ResourceAction_UPDATE_RESOURCE)
	expectedEvent := &central.SensorEvent{
		Id:     "b1",
		Action: central.ResourceAction_UPDATE_RESOURCE,
		Resource: &central.SensorEvent_Binding{
			Binding: &storage.K8SRoleBinding{
				Id:        "b1",
				Name:      "b1",
				Namespace: "n1",
				CreatedAt: protoconv.ConvertTimeToTimestamp(bindings[0].GetCreationTimestamp().Time),
				Subjects:  []*storage.Subject{},
			},
		},
	}
	require.Len(t, event, 1)
	assert.Equal(t, expectedEvent, event[0])
	assert.Equal(t, map[namespacedRoleRef]map[string]*storage.K8SRoleBinding{
		roleBindingRefToNamespaceRef(bindings[0]): {
			"b1": toRoxRoleBinding(bindings[0]),
		},
	}, tested.roleRefToBindings)

	// Upsert the role for the previous binding. We should get the role update and the binding ID should be updated
	event = dispatcher.ProcessEvent(roles[0], nil, central.ResourceAction_CREATE_RESOURCE)
	expectedEvent = &central.SensorEvent{
		Id:     "r1",
		Action: central.ResourceAction_CREATE_RESOURCE,
		Resource: &central.SensorEvent_Role{
			Role: &storage.K8SRole{
				Id:        "r1",
				Name:      "r1",
				Namespace: "n1",
				CreatedAt: protoconv.ConvertTimeToTimestamp(roles[0].GetCreationTimestamp().Time),
				Rules:     []*storage.PolicyRule{},
			},
		},
	}
	require.Len(t, event, 1)
	assert.Equal(t, expectedEvent, event[0])
	// Verify that the role id of the binding that corresponds to this role is now updated
	assert.Equal(t, "r1", tested.bindingsByID["b1"].GetRoleId())
	// check the namespace role ref
	binding0 := toRoxRoleBinding(bindings[0])
	binding0.RoleId = "r1"
	assert.Equal(t, map[namespacedRoleRef]map[string]*storage.K8SRoleBinding{
		roleBindingRefToNamespaceRef(bindings[0]): {
			"b1": binding0,
		},
	}, tested.roleRefToBindings)

	// Add another binding for the first role. Since the role is now present, we should only get the binding update.
	event = dispatcher.ProcessEvent(bindings[1], nil, central.ResourceAction_UPDATE_RESOURCE)
	expectedEvent = &central.SensorEvent{
		Id:     "b2",
		Action: central.ResourceAction_UPDATE_RESOURCE,
		Resource: &central.SensorEvent_Binding{
			Binding: &storage.K8SRoleBinding{
				Id:        "b2",
				Name:      "b2",
				Namespace: "n1",
				RoleId:    "r1",
				CreatedAt: protoconv.ConvertTimeToTimestamp(bindings[1].GetCreationTimestamp().Time),
				Subjects:  []*storage.Subject{},
			},
		},
	}
	require.Len(t, event, 1)
	assert.Equal(t, expectedEvent, event[0])
	// check the namespace role ref
	binding1 := toRoxRoleBinding(bindings[1])
	binding1.RoleId = "r1"
	assert.Equal(t, map[namespacedRoleRef]map[string]*storage.K8SRoleBinding{
		roleBindingRefToNamespaceRef(bindings[0]): {
			"b1": binding0,
			"b2": binding1,
		},
	}, tested.roleRefToBindings)

	// Add a cluster binding with no role, since the role is absent, we should get the update with no role id.
	event = dispatcher.ProcessEvent(clusterBindings[0], nil, central.ResourceAction_CREATE_RESOURCE)
	expectedEvent = &central.SensorEvent{
		Id:     "b3",
		Action: central.ResourceAction_CREATE_RESOURCE,
		Resource: &central.SensorEvent_Binding{
			Binding: &storage.K8SRoleBinding{ // No role ID since the role does not yet exist.
				Id:          "b3",
				Name:        "b3",
				Namespace:   "n1",
				ClusterRole: true,
				CreatedAt:   protoconv.ConvertTimeToTimestamp(clusterBindings[0].GetCreationTimestamp().Time),
				Subjects:    []*storage.Subject{},
			},
		},
	}
	require.Len(t, event, 1)
	assert.Equal(t, expectedEvent, event[0])
	clusterRoleBinding0 := toRoxClusterRoleBinding(clusterBindings[0])
	assert.Equal(t, map[namespacedRoleRef]map[string]*storage.K8SRoleBinding{
		roleBindingRefToNamespaceRef(bindings[0]): {
			"b1": binding0,
			"b2": binding1,
		},
		clusterRoleBindingRefToNamespaceRef(clusterBindings[0]): {
			"b3": clusterRoleBinding0,
		},
	}, tested.roleRefToBindings)

	// Once we upsert the role for the previous binding, we should get the role update and the binding update with the
	// role id filled in.
	event = dispatcher.ProcessEvent(clusterRoles[0], nil, central.ResourceAction_UPDATE_RESOURCE)
	expectedEvent = &central.SensorEvent{
		Id:     "r2",
		Action: central.ResourceAction_UPDATE_RESOURCE,
		Resource: &central.SensorEvent_Role{
			Role: &storage.K8SRole{
				Id:          "r2",
				Name:        "r2",
				Namespace:   "n1",
				ClusterRole: true,
				CreatedAt:   protoconv.ConvertTimeToTimestamp(clusterRoles[0].GetCreationTimestamp().Time),
				Rules:       []*storage.PolicyRule{},
			},
		},
	}
	require.Len(t, event, 1)
	assert.Equal(t, expectedEvent, event[0])
	assert.Equal(t, "r2", tested.bindingsByID["b3"].GetRoleId())

	clusterRoleBinding0.RoleId = "r2"
	assert.Equal(t, map[namespacedRoleRef]map[string]*storage.K8SRoleBinding{
		roleBindingRefToNamespaceRef(bindings[0]): {
			"b1": binding0,
			"b2": binding1,
		},
		clusterRoleBindingRefToNamespaceRef(clusterBindings[0]): {
			"b3": clusterRoleBinding0,
		},
	}, tested.roleRefToBindings)

	// Remove the role. The role should get removed and the binding should get updated with an empty role id.
	event = dispatcher.ProcessEvent(clusterRoles[0], nil, central.ResourceAction_REMOVE_RESOURCE)
	expectedEvent = &central.SensorEvent{
		Id:     "r2",
		Action: central.ResourceAction_REMOVE_RESOURCE,
		Resource: &central.SensorEvent_Role{
			Role: &storage.K8SRole{
				Id:          "r2",
				Name:        "r2",
				Namespace:   "n1",
				ClusterRole: true,
				CreatedAt:   protoconv.ConvertTimeToTimestamp(clusterRoles[0].GetCreationTimestamp().Time),
				Rules:       []*storage.PolicyRule{},
			},
		},
	}

	require.Len(t, event, 1)
	assert.Equal(t, expectedEvent, event[0])
	assert.Equal(t, "", tested.bindingsByID["b3"].GetRoleId())

	clusterRoleBinding0.RoleId = ""
	assert.Equal(t, map[namespacedRoleRef]map[string]*storage.K8SRoleBinding{
		roleBindingRefToNamespaceRef(bindings[0]): {
			"b1": binding0,
			"b2": binding1,
		},
		clusterRoleBindingRefToNamespaceRef(clusterBindings[0]): {
			"b3": clusterRoleBinding0,
		},
	}, tested.roleRefToBindings)

	// Re-add the role. The role should get updated and the binding should be updated back the with role id.
	event = dispatcher.ProcessEvent(clusterRoles[0], nil, central.ResourceAction_UPDATE_RESOURCE)
	expectedEvent = &central.SensorEvent{
		Id:     "r2",
		Action: central.ResourceAction_UPDATE_RESOURCE,
		Resource: &central.SensorEvent_Role{
			Role: &storage.K8SRole{
				Id:          "r2",
				Name:        "r2",
				Namespace:   "n1",
				ClusterRole: true,
				CreatedAt:   protoconv.ConvertTimeToTimestamp(clusterRoles[0].GetCreationTimestamp().Time),
				Rules:       []*storage.PolicyRule{},
			},
		},
	}
	require.Len(t, event, 1)
	assert.Equal(t, expectedEvent, event[0])
	assert.Equal(t, "r2", tested.bindingsByID["b3"].GetRoleId())
	clusterRoleBinding0.RoleId = "r2"
	assert.Equal(t, map[namespacedRoleRef]map[string]*storage.K8SRoleBinding{
		roleBindingRefToNamespaceRef(bindings[0]): {
			"b1": binding0,
			"b2": binding1,
		},
		clusterRoleBindingRefToNamespaceRef(clusterBindings[0]): {
			"b3": clusterRoleBinding0,
		},
	}, tested.roleRefToBindings)

	// Change the binding on b2 to bind to the cluster role.
	bindings[1].RoleRef = v1.RoleRef{
		Name:     "r2",
		Kind:     "ClusterRole",
		APIGroup: "rbac.authorization.k8s.io",
	}
	event = dispatcher.ProcessEvent(bindings[1], nil, central.ResourceAction_UPDATE_RESOURCE)
	expectedEvent = &central.SensorEvent{
		Id:     "b2",
		Action: central.ResourceAction_UPDATE_RESOURCE,
		Resource: &central.SensorEvent_Binding{
			Binding: &storage.K8SRoleBinding{
				Id:        "b2",
				Name:      "b2",
				Namespace: "n1",
				RoleId:    "r2",
				CreatedAt: protoconv.ConvertTimeToTimestamp(bindings[1].GetCreationTimestamp().Time),
				Subjects:  []*storage.Subject{},
			},
		},
	}
	require.Len(t, event, 1)
	assert.Equal(t, expectedEvent, event[0])
	assert.Equal(t, "r2", tested.bindingsByID["b2"].GetRoleId())

	binding1.RoleId = "r2"
	assert.Equal(t, map[namespacedRoleRef]map[string]*storage.K8SRoleBinding{
		roleBindingRefToNamespaceRef(bindings[0]): {
			"b1": binding0,
		},
		clusterRoleBindingRefToNamespaceRef(clusterBindings[0]): {
			"b2": binding1,
			"b3": clusterRoleBinding0,
		},
	}, tested.roleRefToBindings)

	// Removing the binding should just cause a single remove event.
	event = dispatcher.ProcessEvent(bindings[0], nil, central.ResourceAction_REMOVE_RESOURCE)
	expectedEvent = &central.SensorEvent{
		Id:     "b1",
		Action: central.ResourceAction_REMOVE_RESOURCE,
		Resource: &central.SensorEvent_Binding{
			Binding: &storage.K8SRoleBinding{
				Id:        "b1",
				Name:      "b1",
				Namespace: "n1",
				CreatedAt: protoconv.ConvertTimeToTimestamp(bindings[0].GetCreationTimestamp().Time),
				RoleId:    "r1",
				Subjects:  []*storage.Subject{},
			},
		},
	}
	require.Len(t, event, 1)
	assert.Equal(t, expectedEvent, event[0])
	assert.Equal(t, map[namespacedRoleRef]map[string]*storage.K8SRoleBinding{
		roleBindingRefToNamespaceRef(bindings[0]): {},
		clusterRoleBindingRefToNamespaceRef(clusterBindings[0]): {
			"b2": binding1,
			"b3": clusterRoleBinding0,
		},
	}, tested.roleRefToBindings)
}

func BenchmarkRBACUpdater(b *testing.B) {
	for n := 0; n < b.N; n++ {
		// Create a new store and fill it with data
		store := NewStore()
		for i := 0; i < 700; i++ {
			store.UpsertRole(&v1.Role{
				ObjectMeta: metav1.ObjectMeta{
					Name:      fmt.Sprintf("role%d", i),
					Namespace: fmt.Sprintf("namespace%d", i%10),
					UID:       types.UID(uuid.NewV4().String()),
				},
			})
		}
		for i := 0; i < 11572; i++ {
			store.UpsertBinding(&v1.RoleBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name:      fmt.Sprintf("role%d", i),
					Namespace: fmt.Sprintf("namespace%d", i%10),
					UID:       types.UID(uuid.NewV4().String()),
				},
				RoleRef: v1.RoleRef{
					Name: fmt.Sprintf("role%d", i%700),
				},
			})
		}
		// Evaluate permissions
		store.GetPermissionLevelForDeployment(&storage.Deployment{})
	}
}

func BenchmarkRBACUpsertExistingBinding(b *testing.B) {
	b.StopTimer()
	store := NewStore()
	binding := &v1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "role",
			Namespace: "namespace",
			UID:       types.UID(uuid.NewV4().String()),
		},
		RoleRef: v1.RoleRef{
			Name: "role",
		},
	}
	store.UpsertBinding(binding)
	b.StartTimer()
	for n := 0; n < b.N; n++ {
		store.UpsertBinding(binding)
	}
}

func TestStoreGetPermissionLevelForDeployment(t *testing.T) {
	// This test creates roles and bindings and then updates them to match following state:
	// Roles:
	//  1. role-admin (all verbs on all resources)
	//  2. role-default (get)
	// Bindings:
	//  1. admin-subject -> role-admin
	//  2. default-subject -> role-default
	// Cluster Roles:
	//  1. cluster-admin (all verbs on all resources)
	//  2. cluster-elevated (get on all resources)
	// Cluster Bindings:
	//  1. cluster-admin-subject -> cluster-admin
	//  2. cluster-elevated-subject -> cluster-elevated
	roles := []*v1.Role{
		{
			ObjectMeta: meta("role-admin"),
		},
		{
			ObjectMeta: meta("role-default"),
		},
		{
			ObjectMeta: meta("role-admin"),
			Rules: []v1.PolicyRule{{
				APIGroups: []string{""},
				Resources: []string{"*"},
				Verbs:     []string{"*"},
			}},
		},
		{
			ObjectMeta: meta("role-default"),
			Rules: []v1.PolicyRule{{
				APIGroups: []string{""},
				Resources: []string{""},
				Verbs:     []string{"get"},
			}},
		},
	}
	bindings := []*v1.RoleBinding{
		{
			ObjectMeta: meta("b1"),
			RoleRef:    role("role-admin"),
		},
		{
			ObjectMeta: meta("b2"),
			RoleRef:    role("role-default"),
		},
		{
			ObjectMeta: meta("b1"),
			RoleRef:    role("role-admin"),
			Subjects: []v1.Subject{
				{
					Name:      "admin-subject",
					Kind:      v1.ServiceAccountKind,
					Namespace: "n1",
				},
				{
					Name:      "cluster-namespace-subject",
					Kind:      v1.ServiceAccountKind,
					Namespace: "n1",
				},
			},
		},
		{
			ObjectMeta: meta("b2"),
			RoleRef:    role("role-default"),
			Subjects: []v1.Subject{{
				Name:      "default-subject",
				Kind:      v1.ServiceAccountKind,
				Namespace: "n1",
			}},
		},
	}
	clusterRoles := []*v1.ClusterRole{
		{
			ObjectMeta: meta("cluster-admin"),
		},
		{
			ObjectMeta: meta("cluster-elevated"),
		},
		{
			ObjectMeta: meta("cluster-admin"),

			Rules: []v1.PolicyRule{{
				APIGroups: []string{""},
				Resources: []string{"*"},
				Verbs:     []string{"*"},
			}},
		},
		{
			ObjectMeta: meta("cluster-elevated"),
			Rules: []v1.PolicyRule{{
				APIGroups: []string{""},
				Resources: []string{"*"},
				Verbs:     []string{"get"},
			}},
		},
		{
			ObjectMeta: meta("cluster-elevated"),
			Rules: []v1.PolicyRule{{
				APIGroups: []string{""},
				Resources: []string{"*"},
				Verbs:     []string{"get"},
			}},
		},
	}
	clusterBindings := []*v1.ClusterRoleBinding{
		{
			ObjectMeta: meta("b3"),
			RoleRef:    clusterRole("cluster-admin"),
		},
		{
			ObjectMeta: meta("b4"),
			RoleRef:    clusterRole("cluster-elevated"),
		},
		{
			ObjectMeta: meta("b3"),
			RoleRef:    clusterRole("cluster-admin"),
			Subjects: []v1.Subject{{
				Name: "cluster-admin-subject",
				Kind: v1.ServiceAccountKind,
			}},
		},
		{
			ObjectMeta: meta("b4"),
			RoleRef:    clusterRole("cluster-elevated"),
			Subjects: []v1.Subject{
				{
					Name:      "cluster-elevated-subject",
					Kind:      v1.ServiceAccountKind,
					Namespace: "n1",
				},
				{
					Name:      "cluster-elevated-subject-2",
					Kind:      v1.ServiceAccountKind,
					Namespace: "n1",
				},
				{
					Name:      "cluster-namespace-subject",
					Kind:      v1.ServiceAccountKind,
					Namespace: "n1",
				},
			},
		},
		{
			ObjectMeta: meta("b4"),
			RoleRef:    clusterRole("cluster-elevated"),
			Subjects: []v1.Subject{
				{
					Name:      "cluster-elevated-subject",
					Kind:      v1.ServiceAccountKind,
					Namespace: "n1",
				},
				{
					Name:      "cluster-elevated-subject-2",
					Kind:      v1.ServiceAccountKind,
					Namespace: "n1",
				},
				{
					Name:      "cluster-namespace-subject",
					Kind:      v1.ServiceAccountKind,
					Namespace: "n1",
				},
			},
		},
	}

	testCases := []struct {
		deployment storage.Deployment
		expected   storage.PermissionLevel
	}{
		{expected: storage.PermissionLevel_ELEVATED_CLUSTER_WIDE, deployment: storage.Deployment{ServiceAccount: "cluster-elevated-subject", Namespace: "n1"}},
		{expected: storage.PermissionLevel_ELEVATED_CLUSTER_WIDE, deployment: storage.Deployment{ServiceAccount: "cluster-elevated-subject-2", Namespace: "n1"}},
		{expected: storage.PermissionLevel_ELEVATED_CLUSTER_WIDE, deployment: storage.Deployment{ServiceAccount: "cluster-namespace-subject", Namespace: "n1"}},
		{expected: storage.PermissionLevel_NONE, deployment: storage.Deployment{ServiceAccount: "cluster-elevated-subject"}},
		{expected: storage.PermissionLevel_NONE, deployment: storage.Deployment{ServiceAccount: "cluster-admin-subject", Namespace: "n1"}},
		{expected: storage.PermissionLevel_CLUSTER_ADMIN, deployment: storage.Deployment{ServiceAccount: "cluster-admin-subject"}},
		{expected: storage.PermissionLevel_ELEVATED_IN_NAMESPACE, deployment: storage.Deployment{ServiceAccount: "admin-subject", Namespace: "n1"}},
		{expected: storage.PermissionLevel_DEFAULT, deployment: storage.Deployment{ServiceAccount: "default-subject", Namespace: "n1"}},
		{expected: storage.PermissionLevel_NONE, deployment: storage.Deployment{ServiceAccount: "default-subject"}},
		{expected: storage.PermissionLevel_NONE, deployment: storage.Deployment{ServiceAccount: "admin-subject"}},
	}
	updater := setupUpdater(roles, clusterRoles, bindings, clusterBindings)
	updaterWithNoRoles := setupUpdater(roles, clusterRoles, bindings, clusterBindings)
	for _, r := range roles {
		updaterWithNoRoles.RemoveRole(r)
	}
	for _, r := range clusterRoles {
		updaterWithNoRoles.RemoveClusterRole(r)
	}
	updaterWithNoBindings := setupUpdater(roles, clusterRoles, bindings, clusterBindings)
	for _, b := range bindings {
		updaterWithNoBindings.RemoveBinding(b)
	}
	for _, b := range clusterBindings {
		updaterWithNoBindings.RemoveClusterBinding(b)
	}
	for _, tc := range testCases {
		tc := tc

		name := fmt.Sprintf("%s in namespace %q should have %s permision level",
			tc.deployment.ServiceAccount, tc.deployment.Namespace, tc.expected)
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.expected, updater.GetPermissionLevelForDeployment(&tc.deployment))
		})

		name = fmt.Sprintf("%s in namespace %q should have NO permisions after removing roles but keeping bindings",
			tc.deployment.ServiceAccount, tc.deployment.Namespace)
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, storage.PermissionLevel_NONE, updaterWithNoRoles.GetPermissionLevelForDeployment(&tc.deployment))
		})

		name = fmt.Sprintf("%s in namespace %q should have NO permisions after removing bindings but keeping roles",
			tc.deployment.ServiceAccount, tc.deployment.Namespace)
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, storage.PermissionLevel_NONE, updaterWithNoBindings.GetPermissionLevelForDeployment(&tc.deployment))
		})
	}
}

func role(name string) v1.RoleRef {
	return roleRef(name, "Role")
}

func clusterRole(name string) v1.RoleRef {
	return roleRef(name, "ClusterRole")
}

func roleRef(name, kind string) v1.RoleRef {
	return v1.RoleRef{
		Name: name, Kind: kind, APIGroup: "rbac.authorization.k8s.io",
	}
}

func meta(name string) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Name: name, UID: types.UID(name + "-id"), Namespace: "n1",
	}
}

func setupUpdater(roles []*v1.Role, clusterRoles []*v1.ClusterRole, bindings []*v1.RoleBinding, clusterBindings []*v1.ClusterRoleBinding) Store {
	tested := NewStore()
	for _, r := range roles {
		tested.UpsertRole(r)
	}
	for _, b := range bindings {
		tested.UpsertBinding(b)
	}
	for _, r := range clusterRoles {
		tested.UpsertClusterRole(r)
	}
	for _, b := range clusterBindings {
		tested.UpsertClusterBinding(b)
	}
	return tested
}
