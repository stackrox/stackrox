package env

import (
	"time"
)

var (
	// NetworkBaselineObservationPeriod is the observation period for network baselines.
	NetworkBaselineObservationPeriod = registerDurationSetting("ROX_NETWORK_BASELINE_OBSERVATION_PERIOD", time.Hour)
)
