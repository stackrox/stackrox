package kubernetes

import "github.com/stackrox/rox/pkg/set"

var (
	// SystemNamespaceSet is a frozen set of system-specific namespaces in different orchestrators.
	SystemNamespaceSet = set.NewFrozenStringSet(
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
	)
)
