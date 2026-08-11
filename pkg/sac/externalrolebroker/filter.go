package externalrolebroker

import (
	"slices"
	"strings"

	clusterviewv1alpha1 "github.com/stolostron/cluster-lifecycle-api/clusterview/v1alpha1"
	rbacv1 "k8s.io/api/rbac/v1"
)

// FilterUserPermissionsForSupportedK8sResources filters a list of UserPermission objects
// to return only those that reference resources supported by ACS
// (some core Kubernetes resources, some Kubernetes RBAC resources and Stackrox API resources).
//
// A UserPermission is included if its ClusterRoleDefinition contains at least one PolicyRule
// that references one or more of the supported resources with matching API groups.
func FilterUserPermissionsForSupportedK8sResources(
	permissions []clusterviewv1alpha1.UserPermission,
) []clusterviewv1alpha1.UserPermission {
	filtered := make([]clusterviewv1alpha1.UserPermission, 0, len(permissions))

	for _, permission := range permissions {
		if hasSupportedK8sResources(&permission) {
			filtered = append(filtered, permission)
		}
	}

	return filtered
}

// hasSupportedK8sResources checks if the ClusterRoleDefinition within a UserPermission
// contains rules that reference any of the supported Kubernetes resources.
func hasSupportedK8sResources(permission *clusterviewv1alpha1.UserPermission) bool {
	if permission == nil {
		return false
	}
	return slices.ContainsFunc(
		permission.Status.ClusterRoleDefinition.Rules,
		ruleHasSupportedK8sResource,
	)
}

// ruleHasSupportedK8sResource checks if a PolicyRule includes any of
// the supported Kubernetes resources (a resource here is considered
// with the API group it belongs to).
//
// This function supports wildcards ("*") in the policy rule.
func ruleHasSupportedK8sResource(rule rbacv1.PolicyRule) bool {
	// Handle empty APIGroups - no match
	if len(rule.APIGroups) == 0 {
		return false
	}
	for _, resource := range rule.Resources {
		// Handle wildcard resource
		if resource == "*" {
			// If wildcard, check if any APIGroup in the rule is supported
			for _, apiGroup := range rule.APIGroups {
				if apiGroup == "*" {
					return true
				}
				if supportedK8sAPIGroups.Contains(apiGroup) {
					return true
				}
			}
			continue
		}
		baseResource := resource
		if before, _, found := strings.Cut(resource, "/"); found {
			baseResource = before
		}
		for _, apiGroup := range rule.APIGroups {
			if apiGroup == "*" {
				for _, supportedResource := range supportedK8sResources {
					if strings.HasPrefix(supportedResource, baseResource+".") {
						return true
					}
				}
				continue
			}
			resourceInGroup := strings.Join([]string{baseResource, apiGroup}, ".")
			if _, found := resourceMapping[resourceInGroup]; found {
				return true
			}
		}
	}
	return false
}
