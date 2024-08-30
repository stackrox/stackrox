package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/pkg/version"
)

// EmplaceCollector registers, or re-registers, the given metrics collector.
// Metrics collectors cannot be registered if an identical collector
// already exists. This function first unregisters each collector in case
// one already exists, then registers the replacement.
func EmplaceCollector(collectors ...prometheus.Collector) {
	for _, c := range collectors {
		prometheus.Unregister(c)
		prometheus.MustRegister(c)
	}
}

// GetBuildType returns the build type of the binary for telemetry purposes.
func GetBuildType() string {
	if version.IsReleaseVersion() {
		return "release"
	}
	return "internal"
}
