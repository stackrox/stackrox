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
		Help:      "Total number of worker runs",
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
		Help:      "Whether the worker is currently executing (1) or idle (0)",
	}, []string{"name", "kind"})
)

func init() {
	prometheus.MustRegister(runsTotal, runDuration, errorsTotal, runningGauge)
}
