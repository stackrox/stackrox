package manager

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/pkg/metrics"
)

func init() {
	prometheus.MustRegister(
		scansRunningInParallel,
		scanWatcherActiveTime,
		numWatchers,
	)
}

const (
	coPrefix = "complianceoperator_"
)

var (
	numWatchers = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      coPrefix + "scan_watchers_current",
		Help:      "Current number of scan watchers in central's memory",
	})
	scansRunningInParallel = prometheus.NewHistogram(prometheus.HistogramOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		// Num of scans: 0-10 with resolution of 1, 10-200 with resolution of 10.
		Buckets: append(
			prometheus.LinearBuckets(1, 1, 9),
			prometheus.LinearBuckets(10, 10, 20)...,
		),
		Name: coPrefix + "num_scans_running_in_parallel",
		Help: "Number of observations with a given number of watchers being in unfinished state (still running). " +
			"If the metrics are updated every hour, then the value means how many time-blocks of 1 hour had the given " +
			"(or less) number of scans running in parallel.",
	})
	scanWatcherActiveTime = prometheus.NewHistogram(prometheus.HistogramOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Buckets: append(
			// First 2 minutes every 10s
			prometheus.LinearBuckets(1, 10, 12),
			// Rest every minute until the 40m timeout is covered
			prometheus.LinearBuckets(120, 60, 45)...,
		),
		Name: coPrefix + "scan_watchers_active_time_seconds",
		Help: "How long a scan watcher was active",
	})
)
