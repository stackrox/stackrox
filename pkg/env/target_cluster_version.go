package env

var (
	// TargetClusterVersion influences which version of Secured Cluster Services the Operator deploys
	TargetClusterVersion = RegisterSetting("ROX_TARGET_CLUSTER_VERSION", WithDefault("4.1.0-551-g0c9f674289"))
)
