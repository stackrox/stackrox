package externalrolebroker

import (
	"strings"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
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
		"delete",
		"deletecollection",
		"patch",
		"update",
	)
)

// ConvertClusterRoleToPermissionSet converts the rules within a ClusterRoleDefinition
// to a storage.PermissionSet.
//
// The function maps:
//
//   - Core Kubernetes resources (namespaces, secrets, serviceaccounts) to their ACS equivalents
//
//   - RBAC resources (roles, clusterroles, rolebindings, clusterrolebindings) to their ACS equivalents
//
//   - Stackrox API resources (*.api.stackrox.io) to their ACS equivalents
//
//   - Read verbs (get, list, watch) to storage.Access_READ_ACCESS
//
//   - Write verbs (create, update, patch, delete, deletecollection) to storage,Access_READ_WRITE_ACCESS
//
//   - Wildcard verbs (*) to storage.Access_READ_WRITE_ACCESS (most permissive)
//
// Only supported resources that have ACS equivalents are included in the results.
// The returned PermissionSet has a generated ID and an empty name/description
func ConvertClusterRoleToPermissionSet(
	clusterRoleDef clusterviewv1alpha1.ClusterRoleDefinition,
) *storage.PermissionSet {
	resourceToAccess := make(map[string]storage.Access)

	for _, rule := range clusterRoleDef.Rules {
		if len(rule.APIGroups) == 0 {
			continue
		}
		// PermissionSet cannot represent name-scoped access, so skip these
		// to avoid granting broader access than the rule intended.
		if len(rule.ResourceNames) > 0 {
			continue
		}

		access := computeAccessLevel(rule.Verbs)

		for _, k8sResource := range rule.Resources {
			baseResource := getBaseResourceFromResource(k8sResource)
			for _, apiGroup := range rule.APIGroups {
				grantMatchingResources(resourceToAccess, baseResource, apiGroup, access)
			}
		}
	}

	return &storage.PermissionSet{
		Id:               uuid.NewV4().String(),
		ResourceToAccess: resourceToAccess,
	}
}

// grantMatchingResources finds all resourceMapping entries matching the
// resource/apiGroup pair (supporting wildcards in either dimension) and
// grants the given access level to each.
func grantMatchingResources(
	accessMapping map[string]storage.Access,
	resource string, apiGroup string,
	access storage.Access,
) {
	if resource == rbacWildcard || apiGroup == rbacWildcard {
		for key, acsResource := range resourceMapping {
			mappedResource, mappedGroup, _ := strings.Cut(key, ".")
			if (resource == rbacWildcard || resource == mappedResource) &&
				(apiGroup == rbacWildcard || apiGroup == mappedGroup) {
				grantAccessToResource(accessMapping, acsResource, access)
			}
		}
		return
	}
	if acsResource, found := resourceMapping[qualifiedResource(resource, apiGroup)]; found {
		grantAccessToResource(accessMapping, acsResource, access)
	}
}

func grantAccessToResource(
	accessMapping map[string]storage.Access,
	acsResource permissions.ResourceMetadata,
	access storage.Access,
) {
	key := string(acsResource.GetResource())
	if oldAccess, found := accessMapping[key]; found {
		if oldAccess < access {
			accessMapping[key] = access
		}
	} else {
		accessMapping[key] = access
	}
}

func qualifiedResource(k8sResource string, apiGroup string) string {
	return strings.Join([]string{k8sResource, apiGroup}, ".")
}

func computeAccessLevel(verbs []string) storage.Access {
	hasRead := false
	hasWrite := false

	for _, verb := range verbs {
		// Wildcard grants full access
		if verb == rbacWildcard {
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
	// If no recognized read or write verb, grant NO_ACCESS
	return storage.Access_NO_ACCESS
}
