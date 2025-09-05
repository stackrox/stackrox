package updatecomputer

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/pkg/metrics"
)

func init() {
	prometheus.MustRegister(
		UpdateEvents,
		UpdateEventsGauge,
		periodicCleanupDurationMillis,
	)
}

var (
	UpdateEvents = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "update_computer_update_events_total",
		Help: "Counts the internal update events for the categorizeUpdate method in Categorized updateComputer. " +
			"The 'transition' allows counting the transitions of connections between states 'open' and 'closed'." +
			"Action stores the decision whether a given update was sent to Central.",
	}, []string{"transition", "entity", "action", "reason"})
	UpdateEventsGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "update_computer_update_events_current",
		Help: "Counts the internal update events for the categorizeUpdate method in Categorized updateComputer in a single tick. " +
			"The 'transition' allows counting the transitions of connections between states 'open' and 'closed'. in a given tick." +
			"Action stores the decision whether a given update was sent to Central.",
	}, []string{"transition", "entity", "action", "reason"})
	periodicCleanupDurationMillis = prometheus.NewHistogram(prometheus.HistogramOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "update_computer_periodic_cleanup_duration_ms",
		Help:      "Time in milliseconds taken to perform a single periodic cleanup on the categorized update computer.",
		Buckets:   []float64{10, 25, 50, 100, 250, 500, 1000, 2500, 5000},
	})
)
