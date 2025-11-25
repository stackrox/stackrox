package watcher

import (
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/pkg/metrics"
)

func init() {
	prometheus.MustRegister(
		pollDurationHistogram,
		repositoryCountGauge,
	)
}

var (
	pollDurationHistogram = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: metrics.PrometheusNamespace,
			Subsystem: "base_image_watcher",
			Name:      "poll_duration_seconds",
			Help:      "Time taken to complete a poll cycle",
			Buckets:   prometheus.ExponentialBuckets(0.1, 2, 10), // 0.1s to ~102s
		},
		[]string{"error"},
	)

	repositoryCountGauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: "base_image_watcher",
		Name:      "repositories_total",
		Help:      "Number of base image repositories being watched",
	})
)

func recordPollDuration(seconds float64, err error) {
	pollDurationHistogram.WithLabelValues(fmt.Sprintf("%t", err != nil)).Observe(seconds)
}

func recordRepositoryCount(count int) {
	repositoryCountGauge.Set(float64(count))
}
