package metrics

import "github.com/prometheus/client_golang/prometheus"

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
