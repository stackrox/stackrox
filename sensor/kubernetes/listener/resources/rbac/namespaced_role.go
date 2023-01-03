package rbac

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/k8srbac"
	v1 "k8s.io/api/rbac/v1"
)

type rolePermissionLevel int

const (
	permissionNone rolePermissionLevel = iota
	permissionGetOrWatchSomeResource
	permissionListSomeResource // List is considered higher permissions than get/watch
	permissionWriteSomeResource
	permissionWriteAllResources
)

var coreFields = k8srbac.NewPolicyRuleFieldSet(k8srbac.CoreFields()...)

// We cannot use the name "RoleRef" because it's used by the K8s API.
type namespacedRoleRef struct {
	namespace string
	name      string
}

type namespacedRole struct {
	latestUID       string
	permissionLevel rolePermissionLevel
}

func ruleToRolePermissionLevel(rule *v1.PolicyRule) rolePermissionLevel {
	// Note that this will have references to the v1.PolicyRule, so we need to not take
	// ownership or hold onto the reference beyond the end of this method. This avoids
	// cloning the PolicyRules at the cost of creating a new ruleSet for each rule.
	policyRule := &storage.PolicyRule{
		Verbs:     rule.Verbs,
		Resources: rule.Resources,
		ApiGroups: rule.APIGroups,
		// We do not care about ResourceNames or NonResourceUrls.
	}

	switch {
	case coreFields.Grants(policyRule, k8srbac.EffectiveAdmin):
		return permissionWriteAllResources
	case canWriteAResource(policyRule):
		return permissionWriteSomeResource
	case coreFields.Grants(policyRule, k8srbac.ListAnything):
		return permissionListSomeResource
	case canReadAResource(policyRule):
		return permissionGetOrWatchSomeResource
	default:
		return permissionNone
	}
}

func canWriteAResource(pr *storage.PolicyRule) bool {
	return coreFields.Grants(pr, k8srbac.CreateAnything) ||
		coreFields.Grants(pr, k8srbac.BindAnything) ||
		coreFields.Grants(pr, k8srbac.PatchAnything) ||
		coreFields.Grants(pr, k8srbac.UpdateAnything) ||
		coreFields.Grants(pr, k8srbac.DeleteAnything) ||
		coreFields.Grants(pr, k8srbac.DeletecollectionAnything)
}

func canReadAResource(pr *storage.PolicyRule) bool {
	return coreFields.Grants(pr, k8srbac.GetAnything) ||
		coreFields.Grants(pr, k8srbac.ListAnything) ||
		coreFields.Grants(pr, k8srbac.WatchAnything)
}

func maxPermissionLevel(rules []v1.PolicyRule) rolePermissionLevel {
	permissionLevel := permissionNone

	for i := range rules {
		rule := rules[i]
		if p := ruleToRolePermissionLevel(&rule); p > permissionLevel {
			permissionLevel = p
		}
	}

	return permissionLevel
}

func roleAsRef(role *v1.Role) namespacedRoleRef {
	return namespacedRoleRef{
		namespace: role.GetNamespace(),
		name:      role.GetName(),
	}
}

func roleAsNamespacedRole(role *v1.Role) namespacedRole {
	return namespacedRole{
		latestUID:       string(role.GetUID()),
		permissionLevel: maxPermissionLevel(role.Rules),
	}
}

func clusterRoleAsRef(role *v1.ClusterRole) namespacedRoleRef {
	return namespacedRoleRef{
		namespace: "",
		name:      role.GetName(),
	}
}

func clusterRoleAsNamespacedRole(role *v1.ClusterRole) namespacedRole {
	return namespacedRole{
		latestUID:       string(role.GetUID()),
		permissionLevel: maxPermissionLevel(role.Rules),
	}
}

func roleBindingToNamespacedRoleRef(roleBinding *v1.RoleBinding) (namespacedRoleRef, bool) {
	if roleBinding.RoleRef.Kind == "ClusterRole" {
		return namespacedRoleRef{
			namespace: "",
			name:      roleBinding.RoleRef.Name,
		}, true
	}

	return namespacedRoleRef{
		namespace: roleBinding.GetNamespace(),
		name:      roleBinding.RoleRef.Name,
	}, false
}

func clusterRoleBindingToNamespacedRoleRef(clusterRoleBinding *v1.ClusterRoleBinding) namespacedRoleRef {
	return namespacedRoleRef{
		namespace: "",
		name:      clusterRoleBinding.RoleRef.Name,
	}
}
