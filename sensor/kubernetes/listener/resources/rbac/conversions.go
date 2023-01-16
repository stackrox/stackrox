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
	roxRole := &storage.K8SRole{
		Id:          string(role.GetUID()),
		Name:        role.GetName(),
		Namespace:   role.GetNamespace(),
		Labels:      role.GetLabels(),
		Annotations: role.GetAnnotations(),
		ClusterRole: false,
		CreatedAt:   protoconv.ConvertTimeToTimestamp(role.GetCreationTimestamp().Time),
		Rules:       getPolicyRules(role.Rules),
	}
	return roxRole.Clone() // Clone the labels, annotations, and policy rules.
}

func toRoxClusterRole(role *v1.ClusterRole) *storage.K8SRole {
	roxRole := &storage.K8SRole{
		Id:          string(role.GetUID()),
		Name:        role.GetName(),
		Namespace:   role.GetNamespace(),
		Labels:      role.GetLabels(),
		Annotations: role.GetAnnotations(),
		ClusterRole: true,
		CreatedAt:   protoconv.ConvertTimeToTimestamp(role.GetCreationTimestamp().Time),
		Rules:       getPolicyRules(role.Rules),
	}
	return roxRole.Clone() // Clone the labels, annotations, and policy rules.
}

func toRoxRoleBinding(roleBinding *v1.RoleBinding, roleID string, clusterRole bool) *storage.K8SRoleBinding {
	roxBinding := &storage.K8SRoleBinding{
		Id:          string(roleBinding.GetUID()),
		RoleId:      roleID,
		Name:        roleBinding.GetName(),
		Namespace:   roleBinding.GetNamespace(),
		Labels:      roleBinding.GetLabels(),
		Annotations: roleBinding.GetAnnotations(),
		ClusterRole: clusterRole,
		CreatedAt:   protoconv.ConvertTimeToTimestamp(roleBinding.GetCreationTimestamp().Time),
		Subjects:    getSubjects(roleBinding.Subjects),
	}
	return roxBinding.Clone() // Clone the labels and annotations.
}

func toRoxClusterRoleBinding(clusterRoleBinding *v1.ClusterRoleBinding, roleID string) *storage.K8SRoleBinding {
	roxBinding := &storage.K8SRoleBinding{
		Id:          string(clusterRoleBinding.GetUID()),
		RoleId:      roleID, // may be empty in case the named role referenced in the k8s object could not be found
		Name:        clusterRoleBinding.GetName(),
		Namespace:   clusterRoleBinding.GetNamespace(),
		Labels:      clusterRoleBinding.GetLabels(),
		Annotations: clusterRoleBinding.GetAnnotations(),
		ClusterRole: true,
		CreatedAt:   protoconv.ConvertTimeToTimestamp(clusterRoleBinding.GetCreationTimestamp().Time),
		Subjects:    getSubjects(clusterRoleBinding.Subjects),
	}
	return roxBinding.Clone() // Clone the labels and annotations.
}

// The returned PolicyRules are *shallow copies* of the k8sRules, e.g. k8sRules.Verbs,
// not deep clones.
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
