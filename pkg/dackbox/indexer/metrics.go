package indexer

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/pkg/env"
	pkgMetrics "github.com/stackrox/rox/pkg/metrics"
)

var (
	indexObjectsDeduped = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: pkgMetrics.PrometheusNamespace,
		Subsystem: pkgMetrics.CentralSubsystem.String(),
		Name:      "dackbox_index_objects_deduped",
		Help:      "Number of objects deduped in the indexer",
	})
	indexObjectsIndexed = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: pkgMetrics.PrometheusNamespace,
		Subsystem: pkgMetrics.CentralSubsystem.String(),
		Name:      "dackbox_index_objects_indexed",
		Help:      "Number of objects indexer in the indexer",
	})
)

func init() {
	if !env.PostgresDatastoreEnabled.BooleanSetting() {
		prometheus.MustRegister(
			indexObjectsDeduped,
			indexObjectsIndexed,
		)
	}
}
