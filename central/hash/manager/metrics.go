package manager

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/pkg/metrics"
)

func init() {
	prometheus.MustRegister(dedupingHashSizeGauge, dedupingHashCounterVec)
}

var (
	dedupingHashSizeGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "deduping_hash_size",
		Help:      "Number of persisted deduplication hashes for a cluster at last flush",
	}, []string{"cluster"})

	dedupingHashCounterVec = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "deduping_hash_count",
		Help:      "Counts add/remove operations on deduplication hashes by cluster and resource type",
	}, []string{"cluster", "ResourceType", "Operation"})
)
