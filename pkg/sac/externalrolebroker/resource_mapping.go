package externalrolebroker

import (
	"strings"

	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/set"
)

var (
	resourceMapping = map[string]permissions.ResourceMetadata{
		// Core Kubernetes resources
		"namespaces.":      resources.Namespace,
		"nodes.":           resources.Node,
		"secrets.":         resources.Secret,
		"serviceaccounts.": resources.ServiceAccount,
		// Kubernetes RBAC resources
		"clusterroles.rbac.authorization.k8s.io":        resources.K8sRole,
		"roles.rbac.authorization.k8s.io":               resources.K8sRole,
		"clusterrolebindings.rbac.authorization.k8s.io": resources.K8sRoleBinding,
		"rolebindings.rbac.authorization.k8s.io":        resources.K8sRoleBinding,
		// Direct resource mapping
		"accesses.api.stackrox.io":                         resources.Access,
		"administration.api.stackrox.io":                   resources.Administration,
		"alerts.api.stackrox.io":                           resources.Alert,
		"cves.api.stackrox.io":                             resources.CVE,
		"clusters.api.stackrox.io":                         resources.Cluster,
		"compliance.api.stackrox.io":                       resources.Compliance,
		"deployments.api.stackrox.io":                      resources.Deployment,
		"deploymentextensions.api.stackrox.io":             resources.DeploymentExtension,
		"detection.api.stackrox.io":                        resources.Detection,
		"images.api.stackrox.io":                           resources.Image,
		"imageadministration.api.stackrox.io":              resources.ImageAdministration,
		"integrations.api.stackrox.io":                     resources.Integration,
		"k8ssubjects.api.stackrox.io":                      resources.K8sSubject,
		"networkgraphs.api.stackrox.io":                    resources.NetworkGraph,
		"networkpolicies.api.stackrox.io":                  resources.NetworkPolicy,
		"nodes.api.stackrox.io":                            resources.Node,
		"virtualmachines.api.stackrox.io":                  resources.VirtualMachine,
		"vulnerabilitymanagementapprovals.api.stackrox.io": resources.VulnerabilityManagementApprovals,
		"vulnerabilitymanagementrequests.api.stackrox.io":  resources.VulnerabilityManagementRequests,
		"watchedimages.api.stackrox.io":                    resources.WatchedImage,
		"workflowadministration.api.stackrox.io":           resources.WorkflowAdministration,
	}

	supportedK8sAPIGroups, supportedK8sRawResources = listSupportedK8sGroupsAndResources(resourceMapping)
)

func listSupportedK8sGroupsAndResources(mapping map[string]permissions.ResourceMetadata) (set.StringSet, set.StringSet) {
	groups := set.NewStringSet()
	rawResources := set.NewStringSet()
	for key := range mapping {
		resource, apiGroup, found := strings.Cut(key, ".")
		if !found {
			continue
		}
		groups.Add(apiGroup)
		rawResources.Add(resource)
	}
	return groups, rawResources
}
