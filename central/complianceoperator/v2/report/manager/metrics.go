package manager

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/pkg/metrics"
)

func init() {
	prometheus.MustRegister(
		scansRunningInParallel,
		scanWatcherActiveTimeMinutes,
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
	scanWatcherActiveTimeMinutes = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Buckets:   []float64{0.5, 1, 1.5, 2, 3, 4, 5, 10, 20, 30, 40, 45},
		Name:      coPrefix + "scan_watchers_active_time_minutes",
		Help:      "How long (in minutes) a scan watcher was active. Value of 40m is the default timeout.",
	}, []string{"name"})
)
