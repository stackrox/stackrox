package rbac

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/set"
	v1 "k8s.io/api/rbac/v1"
)

type namespacedBindingID struct {
	name      string
	namespace string
	uid       string
}

type namespacedBinding struct {
	bindingID string
	roleRef   namespacedRoleRef   // The role that the subjects are bound to.
	subjects  []namespacedSubject // The subjects that are bound to the referenced role.
}

func (b *namespacedBindingID) IsClusterBinding() bool {
	return len(b.namespace) == 0
}

func (b *namespacedBinding) Equal(other *namespacedBinding) bool {
	if b == nil || other == nil {
		return b == other
	}
	if b.roleRef != other.roleRef {
		return false
	}
	if len(b.subjects) != len(other.subjects) {
		return false
	}
	subjects := make(set.StringSet, len(b.subjects))
	for _, s := range b.subjects {
		subjects.Add(string(s))
	}
	for _, s := range other.subjects {
		if !subjects.Contains(string(s)) {
			return false
		}
	}
	return true
}

func roleBindingToNamespacedBindingID(roleBinding *v1.RoleBinding) namespacedBindingID {
	return namespacedBindingID{namespace: roleBinding.GetNamespace(), name: roleBinding.GetName(), uid: string(roleBinding.GetUID())}
}

func clusterRoleBindingToNamespacedBindingID(clusterRoleBinding *v1.ClusterRoleBinding) namespacedBindingID {
	return namespacedBindingID{namespace: "", name: clusterRoleBinding.GetName(), uid: string(clusterRoleBinding.GetUID())}
}

// roleBindingToNamespacedBinding returns the namespaced binding from the role binding.
// The boolean value will indicate whether the binding binds a cluster role.
func roleBindingToNamespacedBinding(roleBinding *v1.RoleBinding) (*namespacedBinding, bool) {
	subjects := make([]namespacedSubject, 0, len(roleBinding.Subjects))
	for _, s := range getSubjects(roleBinding.Subjects) {
		// We only need this information for evaluating Deployment permission level,
		// so we can keep only ServiceAccount subjects (Pods cannot run as User or Group).
		if s.Kind == storage.SubjectKind_SERVICE_ACCOUNT {
			subjects = append(subjects, nsSubjectFromSubject(s))
		}
	}
	ref, isClusterRole := roleBindingToNamespacedRoleRef(roleBinding)

	return &namespacedBinding{
		bindingID: string(roleBinding.GetUID()),
		subjects:  subjects,
		roleRef:   ref,
	}, isClusterRole
}

func clusterRoleBindingToNamespacedBinding(clusterRoleBinding *v1.ClusterRoleBinding) *namespacedBinding {
	subjects := make([]namespacedSubject, 0, len(clusterRoleBinding.Subjects))
	for _, s := range getSubjects(clusterRoleBinding.Subjects) {
		// We only need this information for evaluating Deployment permission level,
		// so we can keep only ServiceAccount subjects (Pods cannot run as User or Group).
		if s.Kind == storage.SubjectKind_SERVICE_ACCOUNT {
			subjects = append(subjects, nsSubjectFromSubject(s))
		}
	}
	return &namespacedBinding{
		bindingID: string(clusterRoleBinding.GetUID()),
		subjects:  subjects,
		roleRef:   clusterRoleBindingToNamespacedRoleRef(clusterRoleBinding),
	}
}
