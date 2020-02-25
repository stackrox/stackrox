package resources

import (
	"testing"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func TestRBACUpdater(t *testing.T) {
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

	tested := newRBACUpdater().(*rbacUpdaterImpl)

	// Add a binding with no role, should get a binding update with no role id.
	event := tested.upsertBinding(bindings[0])
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
	assert.Equal(t, expectedEvent, event)
	assert.Equal(t, map[namespacedRoleRef]map[string]*storage.K8SRoleBinding{
		roleBindingRefToNamespaceRef(bindings[0]): {
			"b1": toRoxRoleBinding(bindings[0]),
		},
	}, tested.roleRefToBindings)

	// Upsert the role for the previous binding. We should get the role update and the binding ID should be updated
	event = tested.upsertRole(roles[0])
	expectedEvent = &central.SensorEvent{
		Id:     "r1",
		Action: central.ResourceAction_UPDATE_RESOURCE,
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
	assert.Equal(t, expectedEvent, event)
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
	event = tested.upsertBinding(bindings[1])
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
	assert.Equal(t, expectedEvent, event)
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
	event = tested.upsertClusterBinding(clusterBindings[0])
	expectedEvent = &central.SensorEvent{
		Id:     "b3",
		Action: central.ResourceAction_UPDATE_RESOURCE,
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
	assert.Equal(t, expectedEvent, event)
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
	event = tested.upsertClusterRole(clusterRoles[0])
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
	assert.Equal(t, expectedEvent, event)
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
	event = tested.removeClusterRole(clusterRoles[0])
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

	assert.Equal(t, expectedEvent, event)
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
	event = tested.upsertClusterRole(clusterRoles[0])
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
	assert.Equal(t, expectedEvent, event)
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
	event = tested.upsertBinding(bindings[1])
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
	assert.Equal(t, expectedEvent, event)
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
	event = tested.removeBinding(bindings[0])
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
	assert.Equal(t, expectedEvent, event)
	assert.Equal(t, map[namespacedRoleRef]map[string]*storage.K8SRoleBinding{
		roleBindingRefToNamespaceRef(bindings[0]): {},
		clusterRoleBindingRefToNamespaceRef(clusterBindings[0]): {
			"b2": binding1,
			"b3": clusterRoleBinding0,
		},
	}, tested.roleRefToBindings)
}
