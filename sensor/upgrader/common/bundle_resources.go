package common

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// OrderedBundleResourceTypes is the list of all k8s resource types that are (or were at some point) relevant for the sensor
// bundle. They should be ordered according to the preferred creation order. If there are multiple versions of the
// same resource kind, they should be in ascending order.
// IMPORTANT: Any resource type that is part of a sensor bundle deployment (either in a YAML or indirectly via a
//
//	`kubectl` command invocation) must be listed here, otherwise the auto-upgrade will fail.
//
// NEVER REMOVE ELEMENTS FROM THIS LIST, OTHERWISE UPGRADES MIGHT FAIL IN UNEXPECTED WAYS. The upgrader logic
// automatically detects the resources supported by the server, having GroupVersionKinds that are no longer supported
// hence does not hurt.
var OrderedBundleResourceTypes = []schema.GroupVersionKind{
	// No dependencies on other objects
	{Version: "v1", Kind: "ServiceAccount"},
	{Version: "v1", Kind: "Secret"},
	{Version: "v1", Kind: "ConfigMap"},
	{Group: "policy", Version: "v1beta1", Kind: "PodSecurityPolicy"},
	{Group: "security.openshift.io", Version: "v1", Kind: "SecurityContextConstraints"},
	{Group: "networking.k8s.io", Version: "v1", Kind: "NetworkPolicy"},
	{Group: "monitoring.coreos.com", Version: "v1", Kind: "ServiceMonitor"},
	{Group: "monitoring.coreos.com", Version: "v1", Kind: "PrometheusRule"},

	// Might depend on objects above (e.g., referencing podsecuritypolicies)
	{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "Role"},
	{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "ClusterRole"},

	// Depends on serviceaccounts and (cluster)roles
	{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "RoleBinding"},
	{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "ClusterRoleBinding"},

	// Depends on all of the above
	{Group: "apps", Version: "v1", Kind: "DaemonSet"},
	{Group: "apps", Version: "v1", Kind: "Deployment"},

	// No syntactic dependencies, but semantically depends on deployments and daemonsets
	{Version: "v1", Kind: "Service"},

	// Relative order of these is important; we want to prefer v1 if available
	{Group: "admissionregistration.k8s.io", Version: "v1beta1", Kind: "ValidatingWebhookConfiguration"},
	{Group: "admissionregistration.k8s.io", Version: "v1", Kind: "ValidatingWebhookConfiguration"},

	{Group: "networking.istio.io", Version: "v1alpha3", Kind: "DestinationRule"},
}
