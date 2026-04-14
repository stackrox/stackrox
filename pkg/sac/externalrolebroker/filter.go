package externalrolebroker

import (
	"github.com/stackrox/rox/pkg/set"
	clusterviewv1alpha1 "github.com/stolostron/cluster-lifecycle-api/clusterview/v1alpha1"
	rbacv1 "k8s.io/api/rbac/v1"
)

var (
	// Base Kubernetes resources we're interested in for RBAC management
	baseK8sResources = set.NewFrozenStringSet(
		"namespaces",
		"roles",
		"rolebindings",
		"secrets",
		"serviceaccounts",
	)

	// API groups for the base resources
	// "" = core API group (namespaces, secrets, serviceaccounts)
	// "rbac.authorization.k8s.io" = RBAC API group (roles, rolebindings)
	relevantAPIGroups = set.NewFrozenStringSet(
		"",
		"rbac.authorization.k8s.io",
	)
)

// FilterUserPermissionsForBaseK8sResources filters a list of UserPermission objects to return
// only those that reference base Kubernetes resources (namespace, role, rolebinding, secret, serviceaccount).
//
// A UserPermission is included if its ClusterRoleDefinition contains at least one PolicyRule
// that references one or more of the base Kubernetes resources.
func FilterUserPermissionsForBaseK8sResources(permissions []clusterviewv1alpha1.UserPermission) []clusterviewv1alpha1.UserPermission {
	filtered := make([]clusterviewv1alpha1.UserPermission, 0)

	for _, permission := range permissions {
		if hasBaseK8sResources(&permission) {
			filtered = append(filtered, permission)
		}
	}

	return filtered
}

// hasBaseK8sResources checks if a UserPermission's ClusterRoleDefinition contains rules
// that reference any of the base Kubernetes resources.
func hasBaseK8sResources(permission *clusterviewv1alpha1.UserPermission) bool {
	rules := permission.Status.ClusterRoleDefinition.Rules

	for _, rule := range rules {
		if hasRelevantAPIGroup(rule) && hasBaseK8sResourceInRule(rule) {
			return true
		}
	}

	return false
}

// hasRelevantAPIGroup checks if a PolicyRule applies to one of the relevant API groups.
func hasRelevantAPIGroup(rule rbacv1.PolicyRule) bool {
	// If APIGroups is empty or contains "*", it applies to all groups including our relevant ones
	if len(rule.APIGroups) == 0 {
		return false
	}

	for _, apiGroup := range rule.APIGroups {
		if apiGroup == "*" || relevantAPIGroups.Contains(apiGroup) {
			return true
		}
	}

	return false
}

// hasBaseK8sResourceInRule checks if a PolicyRule includes any of the base Kubernetes resources.
func hasBaseK8sResourceInRule(rule rbacv1.PolicyRule) bool {
	for _, resource := range rule.Resources {
		// Handle wildcard
		if resource == "*" {
			return true
		}

		// Handle subresources (e.g., "secrets/status") - extract base resource
		baseResource := resource
		if idx := indexOf(resource, '/'); idx != -1 {
			baseResource = resource[:idx]
		}

		if baseK8sResources.Contains(baseResource) {
			return true
		}
	}

	return false
}

// indexOf returns the index of the first occurrence of sep in s, or -1 if not found.
func indexOf(s string, sep byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == sep {
			return i
		}
	}
	return -1
}
