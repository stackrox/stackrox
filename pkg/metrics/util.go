package metrics

import (
	"errors"

	pkgErrors "github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
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

// CollectToSlice collects the metrics from the vector and places them in a metric slice.
func CollectToSlice(vec *prometheus.GaugeVec) ([]*dto.Metric, error) {
	metricC := make(chan prometheus.Metric)
	go func() {
		defer close(metricC)
		vec.Collect(metricC)
	}()
	var metricsErrs error
	var metricSlice []*dto.Metric
	for metric := range metricC {
		dtoMetric := &dto.Metric{}
		metricsErrs = errors.Join(metricsErrs, metric.Write(dtoMetric))
		metricSlice = append(metricSlice, dtoMetric)
	}
	return metricSlice, pkgErrors.Wrap(metricsErrs, "collecting metrics for vector")
}

// GetBuildType returns the build type of the binary for telemetry purposes.
func GetBuildType() string {
	if version.IsReleaseVersion() {
		return "release"
	}
	return "internal"
}
