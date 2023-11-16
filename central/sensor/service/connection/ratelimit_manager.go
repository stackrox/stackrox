package connection

import (
	"math"
	"time"

	"github.com/stackrox/rox/central/sensor/service/connection/ratetracker"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/grpc/ratelimit"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sync"
)

const (
	boostInitSyncRateLimit = 50

	defaultTopCandidateClusters = 3
	defaultRatePeriod           = 10 * time.Minute
)

type rateLimitManager struct {
	mutex sync.Mutex

	initSyncMaxSensors int
	initSyncSensors    set.StringSet

	msgRateLimiter ratelimit.RateLimiter
	msgRateTracker ratetracker.ClusterRateTracker
}

// newRateLimitManager creates an rateLimitManager with max sensors
// retrieved from env variable, ensuring it is non-negative.
func newRateLimitManager() *rateLimitManager {
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
		initSyncMaxSensors: maxSensors,
		initSyncSensors:    set.NewStringSet(),

		msgRateLimiter: ratelimit.NewRateLimiter(eventRateLimit, env.CentralSensorMaxEventsThrottleDuration.DurationSetting()),
		msgRateTracker: ratetracker.NewClusterRateTracker(defaultRatePeriod, defaultTopCandidateClusters),
	}
}

func (m *rateLimitManager) addInitSync(clusterID string) bool {
	if m == nil || m.msgRateLimiter == nil || m.msgRateTracker == nil {
		return true
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	if len(m.initSyncSensors) >= m.initSyncMaxSensors {
		return false
	}

	if m.initSyncSensors.Add(clusterID) {
		m.msgRateLimiter.IncreaseLimit(boostInitSyncRateLimit)
	}

	return true
}

func (m *rateLimitManager) throttleMsg(clusterID string) bool {
	if m == nil || m.msgRateLimiter == nil || m.msgRateTracker == nil {
		return false
	}
	log.Warnw("Messages from the cluster are throttled by the message rate limiter", logging.ClusterID(clusterID))

	return m.msgRateLimiter.Limit()
}

func (m *rateLimitManager) limitClusterMsg(clusterID string) bool {
	if m == nil || m.msgRateLimiter == nil || m.msgRateTracker == nil {
		return false
	}

	m.msgRateTracker.ReceiveMsg(clusterID)

	// When the global limit is reached. If we are processing a message from
	// a cluster that is a candidate for limiting, we initially apply message
	// throttling. If this throttling doesn't help the situation, we will
	// return false, which should terminate the connection to the cluster.
	return m.msgRateLimiter.LimitWithoutThrottle() && m.msgRateTracker.IsTopCluster(clusterID) && m.throttleMsg(clusterID)
}

func (m *rateLimitManager) removeCluster(clusterID string) {
	if m == nil || m.msgRateLimiter == nil || m.msgRateTracker == nil {
		return
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.msgRateTracker.Remove(clusterID)
	if m.initSyncSensors.Remove(clusterID) {
		m.msgRateLimiter.DecreaseLimit(boostInitSyncRateLimit)
	}
}
