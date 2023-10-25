package env

var (
	// CentralMaxInitSyncSensors defines maximum number of sensors that are doing initial sync in parallel.
	// Default to 0 (no limit).
	CentralMaxInitSyncSensors = RegisterIntegerSetting("ROX_CENTRAL_MAX_INIT_SYNC_SENSORS", 0)

	// CentralRateLimitPerSecond defines number of allowed requests
	// per second to central from all sources. Default 0 (no limit).
	CentralRateLimitPerSecond = RegisterIntegerSetting("ROX_CENTRAL_RATE_LIMIT_PER_SECOND", 0)
)
