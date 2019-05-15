package kubernetes

import "github.com/stackrox/rox/pkg/set"

var (
	systemNamespaceSet = []string{
		"kube-system",
		"kube-public",

		// Istio
		"istio-system",

		// OpenShift specific namespaces
		"kube-service-catalog",
		"management-infra",
		"openshift",
		"openshift-ansible-service-broker",
		"openshift-console",
		"openshift-infra",
		"openshift-logging",
		"openshift-monitoring",
		"openshift-node",
		"openshift-sdn",
		"openshift-template-service-broker",
		"openshift-web-console",
	}
)

// GetSystemNamespaceSet returns all the namespaces we know are for system services
func GetSystemNamespaceSet() set.StringSet {
	return set.NewStringSet(systemNamespaceSet...)
}
