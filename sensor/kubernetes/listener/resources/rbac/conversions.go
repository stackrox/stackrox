package rbac

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protoconv"
	"google.golang.org/protobuf/proto"
	v1 "k8s.io/api/rbac/v1"
)

func toRoleEvent(role *storage.K8SRole, action central.ResourceAction) *central.SensorEvent {
	se := &central.SensorEvent{}
	se.SetId(role.GetId())
	se.SetAction(action)
	se.SetRole(proto.ValueOrDefault(role.CloneVT()))
	return se
}

func toBindingEvent(binding *storage.K8SRoleBinding, action central.ResourceAction) *central.SensorEvent {
	se := &central.SensorEvent{}
	se.SetId(binding.GetId())
	se.SetAction(action)
	se.SetBinding(proto.ValueOrDefault(binding.CloneVT()))
	return se
}

func toRoxRole(role *v1.Role) *storage.K8SRole {
	roxRole := &storage.K8SRole{}
	roxRole.SetId(string(role.GetUID()))
	roxRole.SetName(role.GetName())
	roxRole.SetNamespace(role.GetNamespace())
	roxRole.SetLabels(role.GetLabels())
	roxRole.SetAnnotations(role.GetAnnotations())
	roxRole.SetClusterRole(false)
	roxRole.SetCreatedAt(protoconv.ConvertTimeToTimestamp(role.GetCreationTimestamp().Time))
	roxRole.SetRules(getPolicyRules(role.Rules))
	return roxRole.CloneVT() // Clone the labels, annotations, and policy rules.
}

func toRoxClusterRole(role *v1.ClusterRole) *storage.K8SRole {
	roxRole := &storage.K8SRole{}
	roxRole.SetId(string(role.GetUID()))
	roxRole.SetName(role.GetName())
	roxRole.SetNamespace(role.GetNamespace())
	roxRole.SetLabels(role.GetLabels())
	roxRole.SetAnnotations(role.GetAnnotations())
	roxRole.SetClusterRole(true)
	roxRole.SetCreatedAt(protoconv.ConvertTimeToTimestamp(role.GetCreationTimestamp().Time))
	roxRole.SetRules(getPolicyRules(role.Rules))
	return roxRole.CloneVT() // Clone the labels, annotations, and policy rules.
}

func toRoxRoleBinding(roleBinding *v1.RoleBinding, roleID string, clusterRole bool) *storage.K8SRoleBinding {
	roxBinding := &storage.K8SRoleBinding{}
	roxBinding.SetId(string(roleBinding.GetUID()))
	roxBinding.SetRoleId(roleID)
	roxBinding.SetName(roleBinding.GetName())
	roxBinding.SetNamespace(roleBinding.GetNamespace())
	roxBinding.SetLabels(roleBinding.GetLabels())
	roxBinding.SetAnnotations(roleBinding.GetAnnotations())
	roxBinding.SetClusterRole(clusterRole)
	roxBinding.SetCreatedAt(protoconv.ConvertTimeToTimestamp(roleBinding.GetCreationTimestamp().Time))
	roxBinding.SetSubjects(getSubjects(roleBinding.Subjects))
	return roxBinding.CloneVT() // Clone the labels and annotations.
}

func toRoxClusterRoleBinding(clusterRoleBinding *v1.ClusterRoleBinding, roleID string) *storage.K8SRoleBinding {
	roxBinding := &storage.K8SRoleBinding{}
	roxBinding.SetId(string(clusterRoleBinding.GetUID()))
	roxBinding.SetRoleId(roleID) // may be empty in case the named role referenced in the k8s object could not be found
	roxBinding.SetName(clusterRoleBinding.GetName())
	roxBinding.SetNamespace(clusterRoleBinding.GetNamespace())
	roxBinding.SetLabels(clusterRoleBinding.GetLabels())
	roxBinding.SetAnnotations(clusterRoleBinding.GetAnnotations())
	roxBinding.SetClusterRole(true)
	roxBinding.SetCreatedAt(protoconv.ConvertTimeToTimestamp(clusterRoleBinding.GetCreationTimestamp().Time))
	roxBinding.SetSubjects(getSubjects(clusterRoleBinding.Subjects))
	return roxBinding.CloneVT() // Clone the labels and annotations.
}

// The returned PolicyRules are *shallow copies* of the k8sRules, e.g. k8sRules.Verbs,
// not deep clones.
func getPolicyRules(k8sRules []v1.PolicyRule) []*storage.PolicyRule {
	rules := make([]*storage.PolicyRule, 0, len(k8sRules))
	for _, rule := range k8sRules {
		pr := &storage.PolicyRule{}
		pr.SetVerbs(rule.Verbs)
		pr.SetResources(rule.Resources)
		pr.SetApiGroups(rule.APIGroups)
		pr.SetResourceNames(rule.ResourceNames)
		pr.SetNonResourceUrls(rule.NonResourceURLs)
		rules = append(rules, pr)
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
		subject2 := &storage.Subject{}
		subject2.SetKind(getSubjectKind(subject.Kind))
		subject2.SetName(subject.Name)
		subject2.SetNamespace(subject.Namespace)
		subjects = append(subjects, subject2)
	}
	return subjects
}
