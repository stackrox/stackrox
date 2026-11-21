package periodicworker

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/pkg/metrics"
)

var (
	workerRunsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "worker_runs_total",
		Help:      "Total number of periodic worker runs by outcome (success/error/timeout/panic)",
	}, []string{"worker", "outcome"})

	workerRunDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "worker_run_duration_seconds",
		Help:      "Duration of periodic worker runs in seconds",
		Buckets:   prometheus.DefBuckets,
	}, []string{"worker"})

	workerLastRun = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "worker_last_run_timestamp_seconds",
		Help:      "Unix timestamp of last run (any outcome)",
	}, []string{"worker"})

	workerLastSuccess = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "worker_last_success_timestamp_seconds",
		Help:      "Unix timestamp of last successful run",
	}, []string{"worker"})

	workerConsecutiveFailures = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "worker_consecutive_failures",
		Help:      "Current streak of consecutive failures (reset to 0 on success)",
	}, []string{"worker"})

	workerInProgress = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "worker_in_progress",
		Help:      "1 while a run is executing, else 0",
	}, []string{"worker"})

	workerRunning = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "worker_running",
		Help:      "1 while the loop goroutine is alive, else 0",
	}, []string{"worker"})
)

func init() {
	prometheus.MustRegister(
		workerRunsTotal,
		workerRunDuration,
		workerLastRun,
		workerLastSuccess,
		workerConsecutiveFailures,
		workerInProgress,
		workerRunning,
	)
}
