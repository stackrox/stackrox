package connection

import (
	"fmt"
	"math"

	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/grpc/ratelimit"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sync"
)

const (
	boostInitSyncRateLimit = 50
)

type rateLimitManager struct {
	mutex sync.Mutex

	maxSensors       int
	initSyncSensors  set.StringSet
	eventRateLimiter ratelimit.RateLimiter
}

// NewRateLimitManager creates an rateLimitManager with max sensors
// retrieved from env variable, ensuring it is non-negative.
func NewRateLimitManager() *rateLimitManager {
	maxSensors := env.CentralMaxInitSyncSensors.IntegerSetting()
	if maxSensors < 0 {
		panic(fmt.Sprintf("Negative number is not allowed for max init sync sensors. Check env variable: %q", env.CentralMaxInitSyncSensors.EnvVar()))
	}

	eventRateLimit := env.CentralSensorMaxEventsPerSecond.IntegerSetting()
	if eventRateLimit < 0 {
		panic(fmt.Sprintf("Negative number is not allowed for rate limit of sensors events. Check env variable: %q", env.CentralSensorMaxEventsPerSecond.EnvVar()))
	}

	// Use MaxInt for unlimited max init sync sensors.
	if maxSensors == 0 {
		maxSensors = math.MaxInt
	}

	return &rateLimitManager{
		maxSensors:       maxSensors,
		initSyncSensors:  set.NewStringSet(),
		eventRateLimiter: ratelimit.NewRateLimiter(eventRateLimit),
	}
}

func (m *rateLimitManager) Add(clusterID string) bool {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if len(m.initSyncSensors) >= m.maxSensors {
		return false
	}

	if m.initSyncSensors.Add(clusterID) {
		m.eventRateLimiter.IncreaseLimit(boostInitSyncRateLimit)
	}

	return true
}

func (m *rateLimitManager) Remove(clusterID string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.initSyncSensors.Remove(clusterID) {
		m.eventRateLimiter.DecreaseLimit(boostInitSyncRateLimit)
	}
}

func (m *rateLimitManager) LimitMsg() bool {
	return m.eventRateLimiter.Limit()
}
