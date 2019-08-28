package common

import "k8s.io/apimachinery/pkg/runtime/schema"

// BundleResourceTypes is the list of all k8s resource types that are (or were at some point) relevant for the sensor
// bundle.
// IMPORTANT: Any resource type that is part of a sensor bundle deployment (either in a YAML or indirectly via a
//            `kubectl` command invocation) must be listed here, otherwise the auto-upgrade will fail.
// NEVER REMOVE ELEMENTS FROM THIS LIST, OTHERWISE UPGRADES MIGHT FAIL IN UNEXPECTED WAYS. The upgrader logic
// automatically detects the resources supported by the server, having GroupVersionKinds that are no longer supported
// hence does not hurt.
var BundleResourceTypes = []schema.GroupVersionKind{
	{Version: "v1", Kind: "Service"},
	{Version: "v1", Kind: "ServiceAccount"},
	{Version: "v1", Kind: "Secret"},
	{Version: "v1", Kind: "ConfigMap"},
	{Group: "apps", Version: "v1beta2", Kind: "DaemonSet"},
	{Group: "extensions", Version: "v1beta1", Kind: "Deployment"},
	{Group: "extensions", Version: "v1beta1", Kind: "PodSecurityPolicy"},
	{Group: "admissionregistration.k8s.io", Version: "v1beta1", Kind: "ValidatingWebhookConfiguration"},
	{Group: "networking.k8s.io", Version: "v1", Kind: "NetworkPolicy"},
	{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "ClusterRole"},
	{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "ClusterRoleBinding"},
	{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "Role"},
	{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "RoleBinding"},
	{Group: "security.openshift.io", Version: "v1", Kind: "SecurityContextConstraints"},
}
