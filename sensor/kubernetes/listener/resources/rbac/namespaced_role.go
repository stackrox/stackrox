package rbac

import (
	"github.com/stackrox/rox/generated/storage"
	v1 "k8s.io/api/rbac/v1"
)

// We cannot use the name "RoleRef" because it's used by the K8s API.
type namespacedRoleRef struct {
	namespace string
	name      string
}

type namespacedRole struct {
	latestUID string
	rules     []*storage.PolicyRule
}

func (r *namespacedRoleRef) IsClusterRole() bool {
	return len(r.namespace) == 0
}

func roleAsRef(role *v1.Role) namespacedRoleRef {
	return namespacedRoleRef{
		namespace: role.GetNamespace(),
		name:      role.GetName(),
	}
}

func roleAsNamespacedRole(role *v1.Role) *namespacedRole {
	return &namespacedRole{
		latestUID: string(role.GetUID()),
		rules:     clonePolicyRules(role.Rules), // Clone the v1.PolicyRule slices
	}
}

func clusterRoleAsRef(role *v1.ClusterRole) namespacedRoleRef {
	return namespacedRoleRef{
		namespace: "",
		name:      role.GetName(),
	}
}

func clusterRoleAsNamespacedRole(role *v1.ClusterRole) *namespacedRole {
	return &namespacedRole{
		latestUID: string(role.GetUID()),
		rules:     clonePolicyRules(role.Rules), // Clone the v1.PolicyRule slices
	}
}

func roleBindingToNamespacedRoleRef(roleBinding *v1.RoleBinding) namespacedRoleRef {
	if roleBinding.RoleRef.Kind == "ClusterRole" {
		return namespacedRoleRef{
			namespace: "",
			name:      roleBinding.RoleRef.Name,
		}
	}

	return namespacedRoleRef{
		namespace: roleBinding.GetNamespace(),
		name:      roleBinding.RoleRef.Name,
	}
}

func clusterRoleBindingToNamespacedRoleRef(clusterRoleBinding *v1.ClusterRoleBinding) namespacedRoleRef {
	return namespacedRoleRef{
		namespace: "",
		name:      clusterRoleBinding.RoleRef.Name,
	}
}
