package externalrolebroker

import (
	"slices"
	"strings"

	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/set"
)

var (
	resourceMapping = map[string]permissions.ResourceMetadata{
		// Core Kubernetes resources
		"namespaces.":      resources.Namespace,
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
		"k8sroles.api.stackrox.io":                         resources.K8sRole,
		"k8srolebindings.api.stackrox.io":                  resources.K8sRoleBinding,
		"k8ssubjects.api.stackrox.io":                      resources.K8sSubject,
		"namespaces.api.stackrox.io":                       resources.Namespace,
		"networkgraphs.api.stackrox.io":                    resources.NetworkGraph,
		"networkpolicies.api.stackrox.io":                  resources.NetworkPolicy,
		"nodes.api.stackrox.io":                            resources.Node,
		"secrets.api.stackrox.io":                          resources.Secret,
		"serviceaccounts.api.stackrox.io":                  resources.ServiceAccount,
		"virtualmachines.api.stackrox.io":                  resources.VirtualMachine,
		"vulnerabilitymanagementapprovals.api.stackrox.io": resources.VulnerabilityManagementApprovals,
		"vulnerabilitymanagementrequests.api.stackrox.io":  resources.VulnerabilityManagementRequests,
		"watchedimages.api.stackrox.io":                    resources.WatchedImage,
		"workflowadministration.api.stackrox.io":           resources.WorkflowAdministration,
	}

	supportedK8sAPIGroups = listSupportedK8sAPIGroups(resourceMapping)

	supportedK8sResources = listSupportedK8sResources(resourceMapping)
)

func listSupportedK8sAPIGroups(mapping map[string]permissions.ResourceMetadata) set.StringSet {
	output := set.NewStringSet()
	for resource := range mapping {
		firstSeparator := strings.Index(resource, ".")
		if firstSeparator == -1 {
			continue
		}
		output.Add(resource[firstSeparator+1:])
	}
	return output
}

func listSupportedK8sResources(mapping map[string]permissions.ResourceMetadata) []string {
	output := make([]string, 0, len(mapping))
	for k := range mapping {
		output = append(output, k)
	}
	slices.Sort(output)
	return output
}