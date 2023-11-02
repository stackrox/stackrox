package env

import "time"

var (
	// CentralMaxInitSyncSensors defines maximum number of sensors that are doing initial sync in parallel.
	// Default to 0 (no limit).
	CentralMaxInitSyncSensors = RegisterIntegerSetting("ROX_CENTRAL_MAX_INIT_SYNC_SENSORS", 0)

	// CentralRateLimitPerSecond defines number of allowed requests
	// per second to central from all sources. Default 0 (no limit).
	CentralRateLimitPerSecond = RegisterIntegerSetting("ROX_CENTRAL_RATE_LIMIT_PER_SECOND", 0)

	// CentralRateLimitThrottleDuration sets the maximum allowed throttle
	// duration when the rate limit is reached. If set under 1s (or 0),
	// requests are immediately rejected. The default value is 10s.
	CentralRateLimitThrottleDuration = registerDurationSetting("ROX_CENTRAL_RATE_LIMIT_THROTTLE_DURATION", 10*time.Second, WithDurationZeroAllowed())

	// CentralSensorMaxEventsPerSecond defines number of maximum number of
	// allowed Sensor messages sent from all connected sensors to Central.
	// Default 0 (no limit).
	CentralSensorMaxEventsPerSecond = RegisterIntegerSetting("ROX_CENTRAL_SENSOR_MAX_EVENTS_PER_SECOND", 0)

	// CentralSensorMaxEventsThrottleDuration sets the maximum allowed throttle
	// duration when the global rate limit for sensor message is reached. If
	// set under 1s (or 0), connection is immediately terminate after limit is
	// reached.The default value is 2s.
	CentralSensorMaxEventsThrottleDuration = registerDurationSetting("ROX_CENTRAL_SENSOR_MAX_EVENTS_THROTTLE_DURATION", 2*time.Second, WithDurationZeroAllowed())
)
