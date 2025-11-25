package watcher

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/pkg/metrics"
)

func init() {
	prometheus.MustRegister(
		pollDurationHistogram,
		repositoryCountGauge,
		pollErrorsCounter,
	)
}

var (
	pollDurationHistogram = prometheus.NewHistogram(prometheus.HistogramOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: "base_image_watcher",
		Name:      "poll_duration_seconds",
		Help:      "Time taken to complete a poll cycle",
		Buckets:   prometheus.ExponentialBuckets(0.1, 2, 10), // 0.1s to ~102s
	})

	repositoryCountGauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: "base_image_watcher",
		Name:      "repositories_total",
		Help:      "Number of base image repositories being watched",
	})

	pollErrorsCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: "base_image_watcher",
		Name:      "poll_errors_total",
		Help:      "Total number of poll errors by type",
	}, []string{"error_type"})
)

func recordPollDuration(seconds float64) {
	pollDurationHistogram.Observe(seconds)
}

func recordRepositoryCount(count int) {
	repositoryCountGauge.Set(float64(count))
}

func recordPollError(errorType string) {
	pollErrorsCounter.WithLabelValues(errorType).Inc()
}
