package env

var (
	// SecuredClusterNotifiers defines if certain notifiers are configured to send messages through the secured cluster.
	SecuredClusterNotifiers = RegisterBooleanSetting("ROX_SECURED_CLUSTER_NOTIFICATIONS", false)
)
