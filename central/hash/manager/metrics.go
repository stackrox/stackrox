package manager

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/pkg/metrics"
)

func init() {
	prometheus.MustRegister(dedupingHashSizeGauge)
	prometheus.MustRegister(dedupingHashCounterVec)
}

var (
	dedupingHashSizeGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "deduping_hash_size",
		Help:      "Number of hashes in the deduping hashes",
	}, []string{"cluster"})

	dedupingHashCounterVec = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "deduping_hash_count",
		Help:      "Number of elements in removed from the the hash by resource",
	}, []string{"cluster", "ResourceType"})
)
