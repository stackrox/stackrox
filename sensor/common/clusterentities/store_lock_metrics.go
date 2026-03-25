package clusterentities

import (
	"time"

	"github.com/stackrox/rox/sensor/common/clusterentities/metrics"
)

type unlocker interface {
	Unlock()
}

type runlocker interface {
	RUnlock()
}

func unlockWithMetric(mu unlocker, start time.Time, store, operation string) {
	duration := time.Since(start)
	mu.Unlock()
	metrics.ObserveStoreLockHeldDurationWithOperation(store, operation, duration)
}

func runlockWithMetric(mu runlocker, start time.Time, store, operation string) {
	duration := time.Since(start)
	mu.RUnlock()
	metrics.ObserveStoreLockHeldDurationWithOperation(store, operation, duration)
}
