package externalrolebroker

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/uuid"
	clusterviewv1alpha1 "github.com/stolostron/cluster-lifecycle-api/clusterview/v1alpha1"
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

	// Mapping from Kubernetes resource names to ACS resource names
	k8sToACSResourceMap = map[string]string{
		"namespaces":      string(resources.Namespace.GetResource()),
		"roles":           string(resources.K8sRole.GetResource()),
		"rolebindings":    string(resources.K8sRoleBinding.GetResource()),
		"secrets":         string(resources.Secret.GetResource()),
		"serviceaccounts": string(resources.ServiceAccount.GetResource()),
	}
)

// ConvertClusterRoleToPermissionSet converts a ClusterRoleDefinition's Rules to a storage.PermissionSet.
//
// The function maps:
//   - Kubernetes resources (namespaces, roles, rolebindings, secrets, serviceaccounts) to their ACS equivalents
//   - Read verbs (get, list, watch) to storage.Access_READ_ACCESS
//   - Write verbs (create, update, patch, delete, deletecollection) to storage.Access_READ_WRITE_ACCESS
//   - Wildcard verb (*) to storage.Access_READ_WRITE_ACCESS (most permissive)
//
// Only base Kubernetes resources that have ACS equivalents are included in the result.
// The returned PermissionSet has a generated ID and empty name/description.
func ConvertClusterRoleToPermissionSet(clusterRoleDef clusterviewv1alpha1.ClusterRoleDefinition) *storage.PermissionSet {
	resourceToAccess := make(map[string]storage.Access)

	for _, rule := range clusterRoleDef.Rules {
		// Skip rules that don't apply to our relevant API groups
		if !hasRelevantAPIGroup(rule) {
			continue
		}

		// Process each resource in the rule
		for _, k8sResource := range rule.Resources {
			// Handle subresources (e.g., "secrets/status") - extract base resource
			baseResource := k8sResource
			if idx := indexOf(k8sResource, '/'); idx != -1 {
				baseResource = k8sResource[:idx]
			}

			// Get the ACS resource name for this Kubernetes resource
			acsResource, ok := k8sToACSResourceMap[baseResource]
			if !ok {
				// Handle wildcard resource
				if baseResource == "*" {
					// Grant access to all mapped base resources
					for _, acsRes := range k8sToACSResourceMap {
						access := computeAccessLevel(rule.Verbs)
						if access > resourceToAccess[acsRes] {
							resourceToAccess[acsRes] = access
						}
					}
				}
				// Skip resources we don't map
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
