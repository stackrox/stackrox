package features

var (
	// CollectorEbpf is used to enable eBPF
	CollectorEbpf = registerFeature("Enable eBPF Data Collection", "ROX_COLLECTOR_EBPF", false)

	// NetworkPolicyGenerator is the feature flag for enabling the network policy generator.
	NetworkPolicyGenerator = registerFeature("Network Policy Generator", "ROX_NETWORK_POLICY_GENERATOR", true)

	// PerformDeploymentReconciliation controls whether we actually do the reconciliation.
	// It exists while we stabilize the feature.
	// NB: When removing this feature flag, please also remove references to it in .circleci/config.yml
	PerformDeploymentReconciliation = registerFeature("Reconciliation", "ROX_PERFORM_DEPLOYMENT_RECONCILIATION", true)

	// LicenseEnforcement governs whether the product enforces licenses
	// IMPORTANT: When enabling licensing, DO NOT SET THIS FLAG TO TRUE. DELETE IT AND ASSUME IT IS TRUE WHEREEVER IT IS
	// BEING USED.
	LicenseEnforcement = registerFeature("License Enforcement", "ROX_LICENSE_ENFORCEMENT" /* NEVER CHANGE THIS TO TRUE */, false)

	// K8sRBAC is used to enable k8s rbac collection and processing
	K8sRBAC = registerFeature("Enable k8s RBAC objects collection and processing", "ROX_K8S_RBAC", false)
)
