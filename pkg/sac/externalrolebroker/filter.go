package externalrolebroker

import (
	clusterviewv1alpha1 "github.com/stolostron/cluster-lifecycle-api/clusterview/v1alpha1"
	rbacv1 "k8s.io/api/rbac/v1"
)

var (
	// resourceToAPIGroup maps Kubernetes resource names to their API groups.
	// This configuration defines which resources we're interested in for RBAC management.
	// Key: K8s resource name (plural form)
	// Value: API group ("" for core API, "rbac.authorization.k8s.io" for RBAC, "api.stackrox.io" for StackRox resources)
	resourceToAPIGroup = map[string]string{
		// Core Kubernetes resources
		"namespaces":      "",
		"secrets":         "",
		"serviceaccounts": "",
		// RBAC resources
		"roles":               "rbac.authorization.k8s.io",
		"clusterroles":        "rbac.authorization.k8s.io",
		"rolebindings":        "rbac.authorization.k8s.io",
		"clusterrolebindings": "rbac.authorization.k8s.io",
		// StackRox API resources
		"alerts":                           "api.stackrox.io",
		"compliances":                      "api.stackrox.io",
		"deployments":                      "api.stackrox.io",
		"detections":                       "api.stackrox.io",
		"networkgraphs":                    "api.stackrox.io",
		"vulnerabilitymanagementrequests":  "api.stackrox.io",
		"vulnerabilitymanagementapprovals": "api.stackrox.io",
	}
)

// FilterUserPermissionsForBaseK8sResources filters a list of UserPermission objects to return
// only those that reference configured resources (core Kubernetes resources, RBAC resources, and StackRox API resources).
//
// A UserPermission is included if its ClusterRoleDefinition contains at least one PolicyRule
// that references one or more of the configured resources with matching API groups.
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
// that reference any of the configured Kubernetes resources.
func hasBaseK8sResources(permission *clusterviewv1alpha1.UserPermission) bool {
	rules := permission.Status.ClusterRoleDefinition.Rules

	for _, rule := range rules {
		if hasConfiguredResource(rule) {
			return true
		}
	}

	return false
}

// hasConfiguredResource checks if a PolicyRule includes any of the configured resources
// with matching API groups from resourceToAPIGroup.
func hasConfiguredResource(rule rbacv1.PolicyRule) bool {
	// Handle empty APIGroups - no match
	if len(rule.APIGroups) == 0 {
		return false
	}

	for _, resource := range rule.Resources {
		// Handle wildcard resource
		if resource == "*" {
			// If wildcard, check if any of the rule's APIGroups matches our configured groups
			for _, apiGroup := range rule.APIGroups {
				if apiGroup == "*" {
					return true
				}
				// Check if this apiGroup is in our configuration
				for _, configuredAPIGroup := range resourceToAPIGroup {
					if apiGroup == configuredAPIGroup {
						return true
					}
				}
			}
			continue
		}

		// Handle subresources (e.g., "secrets/status") - extract base resource
		baseResource := resource
		if idx := indexOf(resource, '/'); idx != -1 {
			baseResource = resource[:idx]
		}

		// Check if this resource is configured and if the rule's APIGroups match
		expectedAPIGroup, isConfigured := resourceToAPIGroup[baseResource]
		if !isConfigured {
			continue
		}

		// Check if the rule applies to the expected API group for this resource
		for _, apiGroup := range rule.APIGroups {
			if apiGroup == "*" || apiGroup == expectedAPIGroup {
				return true
			}
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
