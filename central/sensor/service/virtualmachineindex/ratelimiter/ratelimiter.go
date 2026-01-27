package ratelimiter

import (
	"strconv"

	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/rate"
	"github.com/stackrox/rox/pkg/sync"
)

const workloadName = "vm_index_report"

var (
	log      = logging.LoggerForModule()
	once     sync.Once
	instance *rate.Limiter

	onClientDisconnectHook func(clusterID string)
)

// Limiter returns a singleton rate limiter for VM index reports.
func Limiter() *rate.Limiter {
	once.Do(func() {
		instance = buildLimiter()
	})
	return instance
}

// ResetLimiterForTest resets the singleton limiter so tests can set env vars deterministically.
func ResetLimiterForTest() {
	once = sync.Once{}
	instance = nil
}

func buildLimiter() *rate.Limiter {
	rateVal, err := strconv.ParseFloat(env.VMIndexReportRateLimit.Setting(), 64)
	if err != nil {
		log.Warnf("Invalid %s value: %v. Using fallback value of 0.3", env.VMIndexReportRateLimit.EnvVar(), err)
		rateVal = 0.3 // default fallback
	}
	bucket := env.VMIndexReportBucketCapacity.IntegerSetting()
	limiter, err := rate.NewLimiter(workloadName, rateVal, bucket)
	if err != nil {
		log.Panicf("Failed to create rate limiter for %s: %v", workloadName, err)
	}
	return limiter
}

// OnClientDisconnect rebalances the limiter when a Sensor disconnects.
func OnClientDisconnect(clusterID string) {
	Limiter().OnClientDisconnect(clusterID)
	if onClientDisconnectHook != nil {
		onClientDisconnectHook(clusterID)
	}
}

// SetOnClientDisconnectHookForTest registers a test-only callback for OnClientDisconnect.
func SetOnClientDisconnectHookForTest(hook func(clusterID string)) {
	onClientDisconnectHook = hook
}
