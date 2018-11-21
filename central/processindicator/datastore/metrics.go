package datastore

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/pkg/metrics"
)

var (
	prunedProcesses = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "pruned_process_indicators",
		Help:      "Number of process indicators removed by pruning",
	})
)

func incrementPrunedProcessesMetric(num int) {
	prunedProcesses.Add(float64(num))
}

func init() {
	prometheus.MustRegister(prunedProcesses)
}
