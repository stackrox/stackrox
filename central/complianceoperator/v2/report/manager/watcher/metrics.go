package watcher

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/pkg/metrics"
)

func init() {
	prometheus.MustRegister(
		watcherFinishType,
	)
}

const (
	coPrefix = "complianceoperator_"
)

var (
	watcherFinishType = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      coPrefix + "scan_watchers_finish_type_total",
		Help:      "How a watcher run has ended",
	}, []string{"name", "type"})
)
