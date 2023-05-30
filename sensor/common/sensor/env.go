package sensor

import (
	"time"

	"github.com/stackrox/rox/pkg/env"
)

const (
	defaultInitialInterval = 10 * time.Minute
	defaultMaxInterval     = time.Minute
)

var (
	connectionRetryInitialIntervalEnv = env.RegisterSetting("ROX_SENSOR_CONNECTION_RETRY_INITIAL_INTERVAL")
	connectionRetryMaxIntervalEnv     = env.RegisterSetting("ROX_SENSOR_CONNECTION_RETRY_MAX_INTERVAL")
)

func connectionRetryInitialInterval() time.Duration {
	return getOrDefault(connectionRetryInitialIntervalEnv, defaultInitialInterval)
}

func connectionRetryMaxInterval() time.Duration {
	return getOrDefault(connectionRetryMaxIntervalEnv, defaultMaxInterval)
}

func getOrDefault(envVar env.Setting, defaultValue time.Duration) time.Duration {
	if envVar.Setting() == "" {
		return defaultValue
	}

	d, err := time.ParseDuration(envVar.Setting())
	if err != nil {
		log.Warnf("parsing env %s duration value (%s): %s",
			envVar.EnvVar(), envVar.Setting(), err)
		return defaultValue
	}
	return d
}
