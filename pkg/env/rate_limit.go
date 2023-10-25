package env

import "time"

var (
	// CentralMaxInitSyncSensors defines maximum number of sensors that are doing initial sync in parallel.
	// Default to 0 (no limit).
	CentralMaxInitSyncSensors = RegisterIntegerSetting("ROX_CENTRAL_MAX_INIT_SYNC_SENSORS", 0)

	// CentralAPIRateLimitPerSecond defines number of allowed API requests
	// per second to central from all sources. Default 0 (no limit).
	CentralAPIRateLimitPerSecond = RegisterIntegerSetting("ROX_CENTRAL_API_RATE_LIMIT_PER_SECOND", 0)

	// CentralRateLimitThrottleDuration sets the maximum allowed throttle
	// duration when the rate limit is reached. If set under 1s (or 0),
	// requests are immediately rejected. The default value is 10s.
	CentralRateLimitThrottleDuration = registerDurationSetting("ROX_CENTRAL_RATE_LIMIT_THROTTLE_DURATION", 10*time.Second, WithDurationZeroAllowed())
)
