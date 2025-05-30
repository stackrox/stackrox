package manager

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/pkg/metrics"
)

func init() {
	prometheus.MustRegister(
		scansRunningInParallel,
		numWatchers,
	)
}

const (
	coPrefix = "complianceoperator_"
)

var (
	numWatchers = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      coPrefix + "scan_watchers_current",
		Help:      "Current number of scan watchers in central's memory",
	}, []string{"status"})
	scansRunningInParallel = prometheus.NewHistogram(prometheus.HistogramOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		// Num of scans: 0-10 with resolution of 1, 10-200 with resolution of 10.
		Buckets: append([]float64{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}, prometheus.LinearBuckets(10, 10, 19)...),
		Name:    coPrefix + "num_scans_running_in_parallel",
		Help:    "Number of scan watchers being in unfinished state representing scans still running",
	})
)
