package env

const defaultClusterInternalPortSetting = ":9095"

var (
	// ClusterInternalPortSetting has the :port or host:port string for listening for cluster-internal server.
	ClusterInternalPortSetting = RegisterSetting("ROX_CLUSTER_INTERNAL_PORT", WithDefault(defaultClusterInternalPortSetting))
)
