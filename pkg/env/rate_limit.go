package env

var (
	// CentralMaxInitSyncSensors defines maximum number of sensors that are doing initial sync in parallel.
	// Default to 0 (no limit).
	CentralMaxInitSyncSensors = RegisterIntegerSetting("ROX_CENTRAL_MAX_INIT_SYNC_SENSORS", 0)

	// CentralAPIRateLimitPerSecond defines number of allowed API requests
	// per second to central from all sources. Default 0 (no limit).
	CentralAPIRateLimitPerSecond = RegisterIntegerSetting("ROX_CENTRAL_API_RATE_LIMIT_PER_SECOND", 0)

	// CentralSensorMaxEventsPerSecond defines number of maximum number of
	// allowed Sensor messages sent from all connected sensors to Central.
	// Default 0 (no limit).
	CentralSensorMaxEventsPerSecond = RegisterIntegerSetting("ROX_CENTRAL_SENSOR_MAX_EVENTS_PER_SECOND", 0)
)
