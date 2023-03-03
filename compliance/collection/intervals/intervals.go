package intervals

import (
	"math/rand"
	"time"

	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	log = logging.LoggerForModule()

	// Used for testing purposes, to mock random values.
	randFloat64 = rand.Float64
)

// NodeScanIntervals generates node scanning intervals using randomized values.
type NodeScanIntervals struct {
	base       time.Duration
	deviation  float64
	initialMax time.Duration
}

// deviateDuration randomly deviates a duration by a given percentage. Example:
// duration of 10s with 5% deviation means a random duration between 5s and 15s.
func deviateDuration(d time.Duration, percentage float64) time.Duration {
	min, max := 1.0-percentage, 1.0+percentage
	dev := randFloat64()*(max-min) + min
	return multiplyDuration(d, dev)
}

// multiplyDuration multiplies a duration by a float64 and returns the resulting
// duration.
func multiplyDuration(d time.Duration, factor float64) time.Duration {
	return time.Duration(float64(time.Second) * d.Seconds() * factor)
}

// NewNodeScanIntervalFromEnv creates node scanning intervals from environment
// variables.
func NewNodeScanIntervalFromEnv() NodeScanIntervals {
	i := NodeScanIntervals{}
	i.base = env.NodeScanningInterval.DurationSetting()
	i.deviation = 0.0
	if env.NodeScanningIntervalDeviation.IntegerSetting() > 0 {
		i.deviation = float64(env.NodeScanningIntervalDeviation.IntegerSetting()) / 100.0
	}
	i.initialMax = env.NodeScanningMaxInitialWait.DurationSetting()
	log.Infof("scanning intervals: base=%s deviation=%.2f initialMax=%s",
		i.base, i.deviation, i.initialMax)
	return i
}

// Initial returns the initial node scanning interval.
func (i *NodeScanIntervals) Initial() time.Duration {
	interval := multiplyDuration(i.initialMax, randFloat64())
	log.Infof("initial scanning in %s", interval)
	return interval
}

// Next returns the next node scanning interval.
func (i *NodeScanIntervals) Next() time.Duration {
	interval := i.base
	if i.deviation > 0 {
		interval = deviateDuration(interval, i.deviation)
	}
	log.Infof("next node scan in %s", interval)
	return interval
}
