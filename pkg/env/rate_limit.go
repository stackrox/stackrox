package env

var (
	// CentralMaxInitSyncSensors defines maximum number of sensors that are doing initial sync in parallel.
	// Default to 0 (no limit).
	CentralMaxInitSyncSensors = RegisterIntegerSetting("ROX_CENTRAL_MAX_INIT_SYNC_SENSORS", 0)

	// CentralApiRateLimitPerSecond defines number of allowed API requests
	// per second to central from all sources. Default 0 (no limit).
	CentralApiRateLimitPerSecond = RegisterIntegerSetting("ROX_CENTRAL_API_RATE_LIMIT_PER_SECOND", 0)
)
