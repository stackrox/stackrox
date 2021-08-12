package rbac

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protoconv"
	v1 "k8s.io/api/rbac/v1"
)

func toRoleEvent(role *storage.K8SRole, action central.ResourceAction) *central.SensorEvent {
	return &central.SensorEvent{
		Id:     role.GetId(),
		Action: action,
		Resource: &central.SensorEvent_Role{
			Role: role.Clone(),
		},
	}
}

func toBindingEvent(binding *storage.K8SRoleBinding, action central.ResourceAction) *central.SensorEvent {
	return &central.SensorEvent{
		Id:     binding.GetId(),
		Action: action,
		Resource: &central.SensorEvent_Binding{
			Binding: binding.Clone(),
		},
	}
}

func toRoxRole(role *v1.Role) *storage.K8SRole {
	return &storage.K8SRole{
		Id:          string(role.GetUID()),
		Name:        role.GetName(),
		Namespace:   role.GetNamespace(),
		ClusterName: role.GetClusterName(),
		Labels:      role.GetLabels(),
		Annotations: role.GetAnnotations(),
		ClusterRole: false,
		CreatedAt:   protoconv.ConvertTimeToTimestamp(role.GetCreationTimestamp().Time),
		Rules:       getPolicyRules(role.Rules),
	}
}

func toRoxClusterRole(role *v1.ClusterRole) *storage.K8SRole {
	return &storage.K8SRole{
		Id:          string(role.GetUID()),
		Name:        role.GetName(),
		Namespace:   role.GetNamespace(),
		ClusterName: role.GetClusterName(),
		Labels:      role.GetLabels(),
		Annotations: role.GetAnnotations(),
		ClusterRole: true,
		CreatedAt:   protoconv.ConvertTimeToTimestamp(role.GetCreationTimestamp().Time),
		Rules:       getPolicyRules(role.Rules),
	}
}

func toRoxRoleBinding(roleBinding *v1.RoleBinding) *storage.K8SRoleBinding {
	return &storage.K8SRoleBinding{
		Id:          string(roleBinding.GetUID()),
		Name:        roleBinding.GetName(),
		Namespace:   roleBinding.GetNamespace(),
		ClusterName: roleBinding.GetClusterName(),
		Labels:      roleBinding.GetLabels(),
		Annotations: roleBinding.GetAnnotations(),
		ClusterRole: false,
		CreatedAt:   protoconv.ConvertTimeToTimestamp(roleBinding.GetCreationTimestamp().Time),
		Subjects:    getSubjects(roleBinding.Subjects),
	}
}

func toRoxClusterRoleBinding(clusterRoleBinding *v1.ClusterRoleBinding) *storage.K8SRoleBinding {
	return &storage.K8SRoleBinding{
		Id:          string(clusterRoleBinding.GetUID()),
		Name:        clusterRoleBinding.GetName(),
		Namespace:   clusterRoleBinding.GetNamespace(),
		ClusterName: clusterRoleBinding.GetClusterName(),
		Labels:      clusterRoleBinding.GetLabels(),
		Annotations: clusterRoleBinding.GetAnnotations(),
		ClusterRole: true,
		CreatedAt:   protoconv.ConvertTimeToTimestamp(clusterRoleBinding.GetCreationTimestamp().Time),
		Subjects:    getSubjects(clusterRoleBinding.Subjects),
	}
}

func getPolicyRules(k8sRules []v1.PolicyRule) []*storage.PolicyRule {
	rules := make([]*storage.PolicyRule, 0, len(k8sRules))
	for _, rule := range k8sRules {
		rules = append(rules, &storage.PolicyRule{
			Verbs:           rule.Verbs,
			Resources:       rule.Resources,
			ApiGroups:       rule.APIGroups,
			ResourceNames:   rule.ResourceNames,
			NonResourceUrls: rule.NonResourceURLs,
		})
	}
	return rules
}

func getSubjectKind(kind string) storage.SubjectKind {
	switch kind {
	case v1.ServiceAccountKind:
		return storage.SubjectKind_SERVICE_ACCOUNT
	case v1.UserKind:
		return storage.SubjectKind_USER
	case v1.GroupKind:
		return storage.SubjectKind_GROUP
	default:
		log.Warnf("unexpected subject kind %s", kind)
		return storage.SubjectKind_SERVICE_ACCOUNT
	}
}

func getSubjects(k8sSubjects []v1.Subject) []*storage.Subject {
	subjects := make([]*storage.Subject, 0, len(k8sSubjects))
	for _, subject := range k8sSubjects {
		subjects = append(subjects, &storage.Subject{
			Kind:      getSubjectKind(subject.Kind),
			Name:      subject.Name,
			Namespace: subject.Namespace,
		})
	}
	return subjects
}

// K8s helpers since roles don't have their own refs (eye-roll).
////////////////////////////////////////////////////////////////

func roleAsRef(role *v1.Role) namespacedRoleRef {
	return namespacedRoleRef{
		roleRef: v1.RoleRef{
			Kind:     "Role",
			Name:     role.GetName(),
			APIGroup: "rbac.authorization.k8s.io",
		},
		namespace: role.GetNamespace(),
	}
}

func clusterRoleAsRef(role *v1.ClusterRole) namespacedRoleRef {
	return namespacedRoleRef{
		roleRef: v1.RoleRef{
			Kind:     "ClusterRole",
			Name:     role.GetName(),
			APIGroup: "rbac.authorization.k8s.io",
		},
		namespace: "",
	}
}

func roleBindingRefToNamespaceRef(rolebinding *v1.RoleBinding) namespacedRoleRef {
	if rolebinding.RoleRef.Kind == "ClusterRole" {
		return namespacedRoleRef{
			roleRef:   rolebinding.RoleRef,
			namespace: "",
		}
	}

	return namespacedRoleRef{
		roleRef:   rolebinding.RoleRef,
		namespace: rolebinding.GetNamespace(),
	}
}

func clusterRoleBindingRefToNamespaceRef(rolebinding *v1.ClusterRoleBinding) namespacedRoleRef {
	return namespacedRoleRef{
		roleRef:   rolebinding.RoleRef,
		namespace: "",
	}
}
