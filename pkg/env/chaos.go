package env

import "time"

var (
	// ChaosIntervalEnv is the variable that specifies the interval in which to kill Central
	ChaosIntervalEnv = registerDurationSetting("ROX_CHAOS_INTERVAL", 10*time.Minute)
)
