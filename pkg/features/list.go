package features

var (
	// CollectorEbpf is used to enable eBPF
	CollectorEbpf = registerFeature("Enable eBPF Data Collection", "ROX_COLLECTOR_EBPF", false)

	// NetworkPolicyGenerator is the feature flag for enabling the network policy generator.
	NetworkPolicyGenerator = registerFeature("Network Policy Generator", "ROX_NETWORK_POLICY_GENERATOR", true)

	// PerformDeploymentReconciliation controls whether we actually do the reconciliation.
	// It exists while we stabilize the feature.
	// NB: When removing this feature flag, please also remove references to it in .circleci/config.yml
	PerformDeploymentReconciliation = registerFeature("Reconciliation", "ROX_PERFORM_DEPLOYMENT_RECONCILIATION", false)
)
