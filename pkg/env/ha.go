package env

var (
	// CentralMode determines which subsystems Central starts (full, reports, cronjob).
	CentralMode = RegisterSetting("ROX_CENTRAL_MODE", WithDefault("full"))

	// MaxConnectionAgeMins sets the maximum age (in minutes) for gRPC connections before recycling.
	MaxConnectionAgeMins = RegisterIntegerSetting("ROX_GRPC_MAX_CONNECTION_AGE_MINUTES", 60)

	// PolicyPollIntervalSecs sets the interval (in seconds) for polling policy updates in HA mode.
	PolicyPollIntervalSecs = RegisterIntegerSetting("ROX_POLICY_POLL_INTERVAL_SECONDS", 1)

	// HAEnabled enables high-availability mode for Central.
	HAEnabled = RegisterBooleanSetting("ROX_HA_ENABLED", false)
)
