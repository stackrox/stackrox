package common

import (
	"github.com/stackrox/rox/pkg/namespaces"
	"github.com/stackrox/rox/pkg/pods"
)

// AllowedNamespaces is a list of namespaces in which the upgrader may create resources.
//
// When adding a resource to a new namespace, add the namespace to this list with a short
// explanation. Then adapt the namespace validation in sensor/upgrader/preflight/namespace.go.
var AllowedNamespaces = []string{
	// Main StackRox namespace. The upgrader lives here.
	pods.GetPodNamespace(),
	// Needed for OpenShift monitoring integration on OpenShift.
	namespaces.KubeSystem,
	// Needed for OpenShift monitoring integration on OpenShift.
	namespaces.OpenShiftMonitoring,
}
