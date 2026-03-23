package alertmanager

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/pkg/metrics"
)

// observeDurationMs returns a function that, when called, observes the elapsed
// time in milliseconds since observeDurationMs was invoked. Intended for use
// with defer:
//
//	defer observeDurationMs(myHistogram)()
func observeDurationMs(h prometheus.Histogram) func() {
	start := time.Now()
	return func() {
		h.Observe(float64(time.Since(start).Milliseconds()))
	}
}

var (
	alertAndNotifyDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "alert_and_notify_duration_ms",
		Help:      "End-to-end duration of AlertAndNotify in milliseconds",
		Buckets:   prometheus.ExponentialBuckets(4, 2, 12),
	})

	alertAndNotifyIncomingCount = prometheus.NewHistogram(prometheus.HistogramOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "alert_and_notify_incoming_count",
		Help:      "Number of incoming alerts per AlertAndNotify call",
		Buckets:   prometheus.ExponentialBuckets(1, 2, 14),
	})

	mergeManyAlertsDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "merge_many_alerts_duration_ms",
		Help:      "Duration of mergeManyAlerts in milliseconds",
		Buckets:   prometheus.ExponentialBuckets(4, 2, 12),
	})

	mergeManyAlertsPreviousCount = prometheus.NewHistogram(prometheus.HistogramOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "merge_many_alerts_previous_count",
		Help:      "Number of previous alerts fetched from DB per mergeManyAlerts call",
		Buckets:   prometheus.ExponentialBuckets(1, 2, 14),
	})

	alertOutcomeTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "alert_outcome_total",
		Help:      "Cumulative count of alert outcomes from mergeManyAlerts",
	}, []string{"outcome"})
)

func recordAlertOutcomes(newCount, updatedCount, resolvedCount int) {
	alertOutcomeTotal.WithLabelValues("new").Add(float64(newCount))
	alertOutcomeTotal.WithLabelValues("updated").Add(float64(updatedCount))
	alertOutcomeTotal.WithLabelValues("resolved").Add(float64(resolvedCount))
}

func init() {
	metrics.EmplaceCollector(
		alertAndNotifyDuration,
		alertAndNotifyIncomingCount,
		mergeManyAlertsDuration,
		mergeManyAlertsPreviousCount,
		alertOutcomeTotal,
	)
}
