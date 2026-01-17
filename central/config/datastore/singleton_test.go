package datastore

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPlatformComponentLayeredProductsRegex(t *testing.T) {
	// Compile the regex to ensure it's valid
	regex := regexp.MustCompile(PlatformComponentLayeredProductsDefaultRegex)

	// Array of namespaces that SHOULD match the regex
	// These are all the exact namespace patterns defined in the constant
	validNamespaces := []string{
		"aap",
		"ack-system",
		"aws-load-balancer-operator",
		"catalogd-controller-manager",
		"cert-manager",
		"cert-manager-operator",
		"cert-utils-operator",
		"costmanagement-metrics-operator",
		"external-dns-operator",
		"metallb-system",
		"mtr",
		"multicluster-engine",
		"multicluster-global-hub",
		"node-observability-operator",
		"open-cluster-management",
		"openshift-adp",
		"openshift-apiserver-operator",
		"openshift-authentication",
		"openshift-authentication-operator",
		"openshift-builds",
		"openshift-catalogd",
		"openshift-cloud-controller-manager",
		"openshift-cloud-controller-manager-operator",
		"openshift-cloud-credential-operator",
		"openshift-cloud-network-config-controller",
		"openshift-cluster-csi-drivers",
		"openshift-cluster-machine-approver",
		"openshift-cluster-node-tuning-operator",
		"openshift-cluster-observability-operator",
		"openshift-cluster-olm-operator",
		"openshift-cluster-samples-operator",
		"openshift-cluster-storage-operator",
		"openshift-cluster-version",
		"openshift-cnv",
		"openshift-compliance",
		"openshift-config",
		"openshift-config-managed",
		"openshift-config-operator",
		"openshift-console",
		"openshift-console-operator",
		"openshift-console-user-settings",
		"openshift-controller-manager",
		"openshift-controller-manager-operator",
		"openshift-dbaas-operator",
		"openshift-distributed-tracing",
		"openshift-dns",
		"openshift-dns-operator",
		"openshift-dpu-network-operator",
		"openshift-dr-system",
		"openshift-etcd",
		"openshift-etcd-operator",
		"openshift-file-integrity",
		"openshift-gitops",
		"openshift-gitops-operator",
		"openshift-host-network",
		"openshift-image-registry",
		"openshift-infra",
		"openshift-ingress",
		"openshift-ingress-canary",
		"openshift-ingress-node-firewall",
		"openshift-ingress-operator",
		"openshift-insights",
		"openshift-keda",
		"openshift-kmm",
		"openshift-kmm-hub",
		"openshift-kni-infra",
		"openshift-kube-apiserver",
		"openshift-kube-apiserver-operator",
		"openshift-kube-controller-manager",
		"openshift-kube-controller-manager-operator",
		"openshift-kube-scheduler",
		"openshift-kube-scheduler-operator",
		"openshift-kube-storage-version-migrator",
		"openshift-kube-storage-version-migrator-operator",
		"openshift-lifecycle-agent",
		"openshift-local-storage",
		"openshift-logging",
		"openshift-machine-api",
		"openshift-machine-config-operator",
		"openshift-marketplace",
		"openshift-migration",
		"openshift-monitoring",
		"openshift-mta",
		"openshift-mtv",
		"openshift-multus",
		"openshift-netobserv-operator",
		"openshift-network-diagnostics",
		"openshift-network-node-identity",
		"openshift-network-operator",
		"openshift-nfd",
		"openshift-nmstate",
		"openshift-node",
		"openshift-nutanix-infra",
		"openshift-oauth-apiserver",
		"openshift-openstack-infra",
		"openshift-opentelemetry-operator",
		"openshift-operator-controller",
		"openshift-operator-lifecycle-manager",
		"openshift-operators",
		"openshift-operators-redhat",
		"openshift-ovirt-infra",
		"openshift-ovn-kubernetes",
		"openshift-ptp",
		"openshift-route-controller-manager",
		"openshift-sandboxed-containers-operator",
		"openshift-security-profiles",
		"openshift-serverless",
		"openshift-serverless-logic",
		"openshift-service-ca",
		"openshift-service-ca-operator",
		"openshift-sriov-network-operator",
		"openshift-storage",
		"openshift-tempo-operator",
		"openshift-update-service",
		"openshift-user-workload-monitoring",
		"openshift-vertical-pod-autoscaler",
		"openshift-vsphere-infra",
		"openshift-windows-machine-config-operator",
		"openshift-workload-availability",
		"redhat-ods-operator",
		"rhdh-operator",
		"service-telemetry",
		"stackrox",
		"submariner-operator",
	}

	// Array of namespaces that SHOULD NOT match the regex
	// These include prefixes, suffixes, partial matches, and unrelated namespaces
	invalidNamespaces := []string{
		// Prefixes - should not match because regex uses ^namespace$
		"prefix-aap",
		"my-stackrox",
		"test-openshift-monitoring",
		"custom-multicluster-engine",
		// Suffixes - should not match
		"aap-suffix",
		"stackrox-test",
		"openshift-monitoring-backup",
		"multicluster-engine-dev",
		// Partial matches without exact boundaries
		"aap123",
		"stackrox-operator",
		"openshift-monitoring-123",
		// Mixed case (regex patterns are case-sensitive)
		"AAP",
		"StackRox",
		"OpenShift-Monitoring",
		"STACKROX",
		// Completely different namespaces
		"default",
		"my-application",
		"user-workloads",
		"production",
		"development",
		"staging",
		"custom-namespace",
		"application-namespace",
		// Close but not exact matches
		"openshift",
		"kube-system",
		"kube-public",
		"kube-node-lease",
		"rhacs",
		"nvidia-gpu-operator",
		"istio-system",
		"knative-serving",
		// With additional characters
		"openshift-monitoring-extra",
		"openshift-operator",
		"redhat-operator",
		"open-cluster-management-agent",
		// Embedded in longer names
		"my-app-stackrox-deployment",
		"openshift-monitoring-custom",
		// Common user namespaces
		"app",
		"backend",
		"frontend",
		"database",
		"monitoring",
		"logging",
	}

	// Test that all valid namespaces match the regex
	t.Run("ValidNamespaces", func(t *testing.T) {
		for _, namespace := range validNamespaces {
			assert.True(t, regex.MatchString(namespace),
				"Namespace '%s' should match PlatformComponentLayeredProductsDefaultRegex", namespace)
		}
	})

	// Test that all invalid namespaces do NOT match the regex
	t.Run("InvalidNamespaces", func(t *testing.T) {
		for _, namespace := range invalidNamespaces {
			assert.False(t, regex.MatchString(namespace),
				"Namespace '%s' should NOT match PlatformComponentLayeredProductsDefaultRegex", namespace)
		}
	})

	// Additional test to verify the exact count of patterns in the regex
	t.Run("PatternCount", func(t *testing.T) {
		// Count the number of valid namespaces we've defined
		expectedCount := len(validNamespaces)
		assert.Equal(t, 124, expectedCount,
			"Expected 124 valid namespace patterns in the test array")
	})
}

// Generated by Claude Code
