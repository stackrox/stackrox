package env

import "time"

var (
	// SlowQueryThreshold determines what should be considered a slow query for logging purposes
	SlowQueryThreshold = registerDurationSetting("ROX_SLOW_QUERY_THRESHOLD", 30*time.Second)
)
