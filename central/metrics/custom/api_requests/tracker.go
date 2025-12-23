package api_requests

import (
	"github.com/stackrox/rox/central/metrics/custom/tracker"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/telemetry/phonehome"
)

var (
	singleton     *tracker.TrackerBase[*finding]
	singletonOnce sync.Once
)

// new creates a new API request tracker using TrackerBase with counter support.
// The tracker is created with a nil generator since it uses real-time counter
// increments.
func new() *tracker.TrackerBase[*finding] {
	return tracker.MakeGlobalTrackerBase(
		"api_request",
		"API requests",
		LazyLabels,
		nil, // nil generator = counter tracker
	)
}

// Singleton returns the global API request tracker instance.
func Singleton() *tracker.TrackerBase[*finding] {
	singletonOnce.Do(func() {
		singleton = new()
	})
	return singleton
}

// RecordRequest records an API request by incrementing the counter.
// This is a convenience wrapper around TrackerBase.IncrementCounter.
func RecordRequest(rp *phonehome.RequestParams) {
	Singleton().IncrementCounter(rp)
}
