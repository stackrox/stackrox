package ratelimit

import (
	"fmt"

	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/sync"
)

type RateLimiterIdentifier string

const (
	APIRateLimiter    RateLimiterIdentifier = "api"
	SensorRateLimiter RateLimiterIdentifier = "sensor"
)

type RateLimiterRegistry map[RateLimiterIdentifier]RateLimiter

var (
	once sync.Once

	registry RateLimiterRegistry
)

func initRegistry() {
	apiRequestLimitPerSec := env.CentralApiRateLimitPerSecond.IntegerSetting()
	if apiRequestLimitPerSec < 0 {
		panic(fmt.Sprintf("Negative number is not allowed for API request rate limit. Check env variable: %q", env.CentralApiRateLimitPerSecond.EnvVar()))
	}

	maxSensorEventsPerSec := env.CentralSensorMaxEventsPerSecond.IntegerSetting()
	if maxSensorEventsPerSec < 0 {
		panic(fmt.Sprintf("Negative number is not allowed for maximum number of Sensor events. Check env variable: %q", env.CentralSensorMaxEventsPerSecond.EnvVar()))
	}

	registry = map[RateLimiterIdentifier]RateLimiter{
		APIRateLimiter:    NewRateLimiter(apiRequestLimitPerSec),
		SensorRateLimiter: NewRateLimiter(maxSensorEventsPerSec),
	}
}

func NewRateLimiterRegistry() RateLimiterRegistry {
	once.Do(initRegistry)

	return registry
}

func (r RateLimiterRegistry) Get(id RateLimiterIdentifier) RateLimiter {
	if limiter, found := r[id]; found {
		return limiter
	}

	return nil
}
