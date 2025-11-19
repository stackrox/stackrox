package updatecomputer

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/pkg/metrics"
)

func init() {
	prometheus.MustRegister(
		UpdateEvents,
		periodicCleanupDurationSeconds,
	)
}

var (
	UpdateEvents = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "update_computer_update_events_total",
		Help: "Counts the internal update events for the categorizeUpdate method in TransitionBased updateComputer. " +
			"The 'transition' allows counting the transitions of connections between states 'open' and 'closed'." +
			"Action stores the decision whether a given update was sent to Central.",
	}, []string{"transition", "entity", "action", "reason"})
	periodicCleanupDurationSeconds = prometheus.NewHistogram(prometheus.HistogramOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "update_computer_periodic_cleanup_duration_seconds",
		Help:      "Time in seconds taken to perform a single periodic cleanup on the transition-based update computer.",
		Buckets:   []float64{0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5},
	})
)
