package rbac

import v1 "k8s.io/api/rbac/v1"

// We cannot use the name "RoleRef" because it's used by the K8s API.
type namespacedRoleRef struct {
	namespace string
	name      string
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

func clusterRoleAsRef(role *v1.ClusterRole) namespacedRoleRef {
	return namespacedRoleRef{
		namespace: "",
		name:      role.GetName(),
	}
}

func roleBindingRefToNamespaceRef(roleBinding *v1.RoleBinding) namespacedRoleRef {
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

func clusterRoleBindingRefToNamespaceRef(clusterRoleBinding *v1.ClusterRoleBinding) namespacedRoleRef {
	return namespacedRoleRef{
		namespace: "",
		name:      clusterRoleBinding.RoleRef.Name,
	}
}
