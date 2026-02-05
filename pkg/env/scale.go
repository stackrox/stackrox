package env

var (
	// ScaleTestEnabled signifies that a scale test is being run
	ScaleTestEnabled = RegisterBooleanSetting("ROX_SCALE_TEST", false)

	// FakeWorkloadStoragePath signifies the path where we should store IDs for the fake workload to avoid reconciliation
	// If unset, then no storage will occur
	FakeWorkloadStoragePath = RegisterSetting("ROX_FAKE_WORKLOAD_STORAGE")

	// RemoteClusterSecretName specifies the name of the secret containing kubeconfig for remote cluster access
	// If set, sensor will use the kubeconfig from this secret to connect to a remote cluster instead of the local cluster
	RemoteClusterSecretName = RegisterSetting("ROX_REMOTE_CLUSTER_SECRET")

	// RemoteClusterSecretNamespace specifies the namespace where the remote cluster secret is located
	// Defaults to the sensor's own namespace if not specified
	RemoteClusterSecretNamespace = RegisterSetting("ROX_REMOTE_CLUSTER_SECRET_NAMESPACE")
)
