package externalrolebroker

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/uuid"
	clusterviewv1alpha1 "github.com/stolostron/cluster-lifecycle-api/clusterview/v1alpha1"
	rbacv1 "k8s.io/api/rbac/v1"
)

var (
	// Kubernetes verbs that map to READ_ACCESS
	readVerbs = set.NewFrozenStringSet(
		"get",
		"list",
		"watch",
	)

	// Kubernetes verbs that map to READ_WRITE_ACCESS
	writeVerbs = set.NewFrozenStringSet(
		"create",
		"update",
		"patch",
		"delete",
		"deletecollection",
	)

	// Mapping from Kubernetes resource names to ACS resource names.
	// Note: Both roles and clusterroles map to K8sRole, and both rolebindings
	// and clusterrolebindings map to K8sRoleBinding, as the distinction is about
	// scope rather than resource type.
	k8sToACSResourceMap = map[string]string{
		// Core Kubernetes resources
		"namespaces":      string(resources.Namespace.GetResource()),
		"secrets":         string(resources.Secret.GetResource()),
		"serviceaccounts": string(resources.ServiceAccount.GetResource()),
		// RBAC resources
		"roles":               "K8sRole",
		"clusterroles":        "K8sRole",
		"rolebindings":        "K8sRoleBinding",
		"clusterrolebindings": "K8sRoleBinding",
		// StackRox API resources
		"alerts":                           string(resources.Alert.GetResource()),
		"compliances":                      string(resources.Compliance.GetResource()),
		"deployments":                      string(resources.Deployment.GetResource()),
		"detections":                       string(resources.Detection.GetResource()),
		"networkgraphs":                    string(resources.NetworkGraph.GetResource()),
		"vulnerabilitymanagementrequests":  string(resources.VulnerabilityManagementRequests.GetResource()),
		"vulnerabilitymanagementapprovals": string(resources.VulnerabilityManagementApprovals.GetResource()),
	}
)

// ConvertClusterRoleToPermissionSet converts a ClusterRoleDefinition's Rules to a storage.PermissionSet.
//
// The function maps:
//   - Core Kubernetes resources (namespaces, secrets, serviceaccounts) to their ACS equivalents
//   - RBAC resources (roles, clusterroles, rolebindings, clusterrolebindings) to their ACS equivalents
//   - StackRox API resources (alerts, compliances, deployments, detections, networkgraphs,
//     vulnerabilitymanagementrequests, vulnerabilitymanagementapprovals) to their ACS equivalents
//   - Read verbs (get, list, watch) to storage.Access_READ_ACCESS
//   - Write verbs (create, update, patch, delete, deletecollection) to storage.Access_READ_WRITE_ACCESS
//   - Wildcard verb (*) to storage.Access_READ_WRITE_ACCESS (most permissive)
//
// Only configured resources that have ACS equivalents are included in the result.
// The returned PermissionSet has a generated ID and empty name/description.
func ConvertClusterRoleToPermissionSet(clusterRoleDef clusterviewv1alpha1.ClusterRoleDefinition) *storage.PermissionSet {
	resourceToAccess := make(map[string]storage.Access)

	for _, rule := range clusterRoleDef.Rules {
		// Skip rules with empty APIGroups
		if len(rule.APIGroups) == 0 {
			continue
		}

		// Process each resource in the rule
		for _, k8sResource := range rule.Resources {
			// Handle subresources (e.g., "secrets/status") - extract base resource
			baseResource := k8sResource
			if idx := indexOf(k8sResource, '/'); idx != -1 {
				baseResource = k8sResource[:idx]
			}

			// Handle wildcard resource
			if baseResource == "*" {
				// Grant access to all configured resources that match the rule's API groups
				for configuredResource, configuredAPIGroup := range resourceToAPIGroup {
					if !ruleMatchesAPIGroup(rule, configuredAPIGroup) {
						continue
					}
					acsResource, ok := k8sToACSResourceMap[configuredResource]
					if !ok {
						continue
					}
					access := computeAccessLevel(rule.Verbs)
					if access > resourceToAccess[acsResource] {
						resourceToAccess[acsResource] = access
					}
				}
				continue
			}

			// Check if this resource is configured
			expectedAPIGroup, isConfigured := resourceToAPIGroup[baseResource]
			if !isConfigured {
				continue
			}

			// Check if the rule's APIGroups match the expected API group for this resource
			if !ruleMatchesAPIGroup(rule, expectedAPIGroup) {
				continue
			}

			// Get the ACS resource name for this Kubernetes resource
			acsResource, ok := k8sToACSResourceMap[baseResource]
			if !ok {
				continue
			}

			// Compute the access level for this resource based on verbs
			access := computeAccessLevel(rule.Verbs)

			// If this resource already has an access level, use the more permissive one
			if existingAccess, exists := resourceToAccess[acsResource]; exists {
				if access > existingAccess {
					resourceToAccess[acsResource] = access
				}
			} else {
				resourceToAccess[acsResource] = access
			}
		}
	}

	return &storage.PermissionSet{
		Id:               uuid.NewV4().String(),
		Name:             "",
		Description:      "",
		ResourceToAccess: resourceToAccess,
	}
}

// ruleMatchesAPIGroup checks if a PolicyRule's APIGroups include the expected API group.
func ruleMatchesAPIGroup(rule rbacv1.PolicyRule, expectedAPIGroup string) bool {
	for _, apiGroup := range rule.APIGroups {
		if apiGroup == "*" || apiGroup == expectedAPIGroup {
			return true
		}
	}
	return false
}

// computeAccessLevel determines the access level based on the list of Kubernetes verbs.
// Returns the most permissive access level found in the verbs.
func computeAccessLevel(verbs []string) storage.Access {
	hasRead := false
	hasWrite := false

	for _, verb := range verbs {
		// Wildcard grants full access
		if verb == "*" {
			return storage.Access_READ_WRITE_ACCESS
		}

		if readVerbs.Contains(verb) {
			hasRead = true
		}
		if writeVerbs.Contains(verb) {
			hasWrite = true
		}
	}

	// If any write verb is present, grant READ_WRITE_ACCESS
	if hasWrite {
		return storage.Access_READ_WRITE_ACCESS
	}

	// If only read verbs are present, grant READ_ACCESS
	if hasRead {
		return storage.Access_READ_ACCESS
	}

	// If no recognized verbs, grant NO_ACCESS
	return storage.Access_NO_ACCESS
}
