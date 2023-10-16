package connection

import (
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
		log.Panicf("Negative number is not allowed for max init sync sensors. Check env variable: %q", env.CentralMaxInitSyncSensors.EnvVar())
	}

	eventRateLimit := env.CentralSensorMaxEventsPerSecond.IntegerSetting()
	if eventRateLimit < 0 {
		log.Panicf("Negative number is not allowed for rate limit of sensors events. Check env variable: %q", env.CentralSensorMaxEventsPerSecond.EnvVar())
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

func (m *rateLimitManager) AddInitSync(clusterID string) bool {
	if m == nil {
		return true
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	if len(m.initSyncSensors) >= m.maxSensors {
		return false
	}

	if m.initSyncSensors.Add(clusterID) && m.eventRateLimiter != nil {
		m.eventRateLimiter.IncreaseLimit(boostInitSyncRateLimit)
	}

	return true
}

func (m *rateLimitManager) RemoveInitSync(clusterID string) {
	if m == nil {
		return
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.initSyncSensors.Remove(clusterID) && m.eventRateLimiter != nil {
		m.eventRateLimiter.DecreaseLimit(boostInitSyncRateLimit)
	}
}

func (m *rateLimitManager) LimitMsg() bool {
	if m == nil || m.eventRateLimiter == nil {
		return false
	}

	return m.eventRateLimiter.Limit()
}
