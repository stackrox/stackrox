package metrics

import (
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/sync"
)

/*
Custom Prometheus metrics are user facing metrics, served on the API endpoint.
The metrics are configured via the private section of the 'config' API.

A metric is configured with a list of labels supported by the metric tracker.

Metrics are immutable. To change anything, the metric has to be deleted and
recreated with separate requests.
*/

var (
	// Those are dynamically defined metrics, configured by users via the system
	// private configuration.
	customAggregatedMetrics sync.Map // [string]*metricRecord

	CustomRegistry = prometheus.NewRegistry()
)

type metricRecord struct {
	*prometheus.GaugeVec
}

// UnregisterCustomAggregatedMetric unregister the given metric by name.
func UnregisterCustomAggregatedMetric(name string) bool {
	v, ok := customAggregatedMetrics.LoadAndDelete(name)
	if !ok {
		return false
	}
	return CustomRegistry.Unregister(v.(*metricRecord).GaugeVec)
}

// RegisterCustomAggregatedMetric registers user-defined aggregated metrics
// according to the system private configuration.
func RegisterCustomAggregatedMetric(name string, category string, period time.Duration, labels []string) error {
	gauge := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      name,
		Help: "The total number of " + category + " aggregated by " + strings.Join(labels, ",") +
			" and gathered every " + period.String(),
	}, labels)

	if _, loaded := customAggregatedMetrics.LoadOrStore(name, &metricRecord{gauge}); loaded {
		return nil
	}
	return CustomRegistry.Register(gauge)
}

// SetCustomAggregatedCount registers the metric vector with the values,
// according to the system private configuration.
func SetCustomAggregatedCount(metricName string, labels prometheus.Labels, total int) {
	if metric, ok := customAggregatedMetrics.Load(metricName); ok {
		metric.(*metricRecord).GaugeVec.With(labels).Set(float64(total))
	}
}
