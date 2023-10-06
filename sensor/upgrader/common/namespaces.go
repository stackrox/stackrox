package common

import "github.com/stackrox/rox/pkg/namespaces"

// AllowedNamespaces is a list of namespaces in which the upgrader may create resources.
//
// When adding a resource to a new namespace, add the namespace to this list with a short
// explanation. Then adapt the namespace validation in sensor/upgrader/preflight/namespace.go.
var AllowedNamespaces = []string{
	// Main StackRox namespace. The upgrader lives here.
	Namespace,
	// Needed for OpenShift monitoring integration on OpenShift.
	namespaces.KubeSystem,
	// Needed for OpenShift monitoring integration on OpenShift.
	namespaces.OpenShiftMonitoring,
}
