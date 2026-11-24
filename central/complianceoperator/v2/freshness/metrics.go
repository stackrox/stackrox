package freshness

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/pkg/metrics"
)

func init() {
	prometheus.MustRegister(
		sweepDuration,
		resultsMarkedStale,
		sweepErrors,
	)
}

var (
	sweepDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: "compliance_freshness_sweeper",
		Name:      "sweep_duration_seconds",
		Help:      "Time to complete one full sweep",
	})

	resultsMarkedStale = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: "compliance_freshness_sweeper",
		Name:      "results_marked_stale_total",
		Help:      "Compliance check results marked stale",
	})

	sweepErrors = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: "compliance_freshness_sweeper",
		Name:      "sweep_errors_total",
		Help:      "Errors encountered during a sweep, by stage",
	}, []string{"stage"})
)
