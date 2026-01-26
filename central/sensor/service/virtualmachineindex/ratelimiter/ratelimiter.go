package ratelimiter

import (
	"strconv"

	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/rate"
)

const workloadName = "vm_index_report"

var (
	log = logging.LoggerForModule()
)

// NewFromEnv returns a rate limiter configured from env settings.
func NewFromEnv() *rate.Limiter {
	return buildLimiter()
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
