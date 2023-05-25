package manager

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/pkg/metrics"
)

func init() {
	prometheus.MustRegister(dedupingHashSizeGauge)
}

var (
	dedupingHashSizeGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "deduping_hash_size",
		Help:      "Number of hashes in the deduping hashes",
	}, []string{"cluster"})
)
