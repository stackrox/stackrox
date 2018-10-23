package enricher

import (
	"github.com/prometheus/client_golang/prometheus"
	pkgMetrics "github.com/stackrox/rox/pkg/metrics"
)

// This interface encapsulates the metrics this package needs.
type metrics interface {
	IncrementMetadataCacheHit()

	IncrementMetadataCacheMiss()

	IncrementScanCacheHit()

	IncrementScanCacheMiss()
}

type metricsImpl struct {
	metadataCacheHits   prometheus.Counter
	metadataCacheMisses prometheus.Counter
	scanCacheHits       prometheus.Counter
	scanCacheMisses     prometheus.Counter
}

func (m *metricsImpl) IncrementMetadataCacheHit() {
	m.metadataCacheHits.Inc()
}

func (m *metricsImpl) IncrementMetadataCacheMiss() {
	m.metadataCacheMisses.Inc()
}

func (m *metricsImpl) IncrementScanCacheHit() {
	m.scanCacheHits.Inc()
}

func (m *metricsImpl) IncrementScanCacheMiss() {
	m.scanCacheMisses.Inc()
}

func newMetrics(subsystem pkgMetrics.Subsystem) metrics {
	m := &metricsImpl{
		metadataCacheHits: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: pkgMetrics.PrometheusNamespace,
			Subsystem: subsystem.String(),
			Name:      "metadata_cache_hits",
			Help:      "Number of cache hits in the metadata cache",
		}),
		metadataCacheMisses: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: pkgMetrics.PrometheusNamespace,
			Subsystem: subsystem.String(),
			Name:      "metadata_cache_misses",
			Help:      "Number of cache misses in the metadata cache",
		}),
		scanCacheHits: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: pkgMetrics.PrometheusNamespace,
			Subsystem: subsystem.String(),
			Name:      "scan_cache_hits",
			Help:      "Number of cache hits in the scan cache",
		}),
		scanCacheMisses: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: pkgMetrics.PrometheusNamespace,
			Subsystem: subsystem.String(),
			Name:      "scan_cache_misses",
			Help:      "Number of cache misses in the scan cache",
		}),
	}

	pkgMetrics.EmplaceCollector(
		m.metadataCacheHits,
		m.metadataCacheMisses,
		m.scanCacheHits,
		m.scanCacheMisses,
	)

	return m
}
