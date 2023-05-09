package env

var (
	// SecuredClusterNotifiers controls whether notifications to certain notifiers are sent through the secured cluster.
	SecuredClusterNotifiers = RegisterBooleanSetting("ROX_SECURED_CLUSTER_NOTIFICATIONS", false)
)
