package backgroundworker

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/pkg/metrics"
)

var (
	runsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.BackgroundWorkerSubsystem.String(),
		Name:      "runs_total",
		Help:      "Total number of worker runs (ticks, flushes, or attempts)",
	}, []string{"name", "kind"})

	runDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.BackgroundWorkerSubsystem.String(),
		Name:      "run_duration_seconds",
		Help:      "Duration of worker runs in seconds",
		Buckets:   prometheus.ExponentialBuckets(0.01, 3, 10),
	}, []string{"name", "kind"})

	errorsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.BackgroundWorkerSubsystem.String(),
		Name:      "errors_total",
		Help:      "Total number of worker run errors",
	}, []string{"name", "kind"})

	runningGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.BackgroundWorkerSubsystem.String(),
		Name:      "running",
		Help:      "Whether the worker is currently executing (1/0 for periodic/once, in-flight count for queue)",
	}, []string{"name", "kind"})
)

func init() {
	prometheus.MustRegister(runsTotal, runDuration, errorsTotal, runningGauge)
}
