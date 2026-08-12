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
		if slices.ContainsFunc(
			permission.Status.ClusterRoleDefinition.Rules,
			ruleHasSupportedK8sResource,
		) {
			filtered = append(filtered, permission)
		}
	}

	return filtered
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
	hasNamespaceResource := false
	hasBaseK8sAPIGroup := false
	for _, resource := range rule.Resources {
		if resource == "*" || resource == "namespaces" {
			hasNamespaceResource = true
			break
		}
	}
	for _, apiGroup := range rule.APIGroups {
		if apiGroup == "*" || apiGroup == "" {
			hasBaseK8sAPIGroup = true
			break
		}
	}
	if len(rule.ResourceNames) > 0 && !hasBaseK8sAPIGroup && !hasNamespaceResource {
		// ACS does not have granularity to apply ResourceName restriction on object access.
		// One exception to this are namespaces, for which UserPermission objects can be considered.
		// Otherwise, the UserPermission object is ignored by ACS.
		return false
	}
	for _, resource := range rule.Resources {
		baseResource := getBaseResourceFromResource(resource)
		for _, apiGroup := range rule.APIGroups {
			if matches(baseResource, apiGroup) {
				return true
			}
		}
	}
	return false
}

func matches(resource string, apiGroup string) bool {
	if resource == "*" && apiGroup == "*" {
		return true
	}
	if apiGroup == "*" {
		if supportedK8sRawResources.Contains(resource) {
			return true
		}
		return false
	}
	if resource == "*" {
		if supportedK8sAPIGroups.Contains(apiGroup) {
			return true
		}
		return false
	}
	resourceWithApiGroup := strings.Join([]string{resource, apiGroup}, ".")
	if _, found := resourceMapping[resourceWithApiGroup]; found {
		return true
	}
	return false
}

func getBaseResourceFromResource(resource string) string {
	if prefix, _, found := strings.Cut(resource, "/"); found {
		return prefix
	}
	return resource
}
