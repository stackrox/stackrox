package enricher

import (
	"fmt"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	pkgMetrics "github.com/stackrox/rox/pkg/metrics"
)

// This interface encapsulates the metrics this package needs.
type metrics interface {
	IncrementMetadataCacheHit()

	IncrementMetadataCacheMiss()

	IncrementScanCacheHit()

	IncrementScanCacheMiss()

	SetScanDurationTime(start time.Time, scanner string, err error)
}

type metricsImpl struct {
	metadataCacheHits   prometheus.Counter
	metadataCacheMisses prometheus.Counter
	scanCacheHits       prometheus.Counter
	scanCacheMisses     prometheus.Counter

	scanTimeDuration *prometheus.HistogramVec
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

func startTimeToMS(t time.Time) float64 {
	return float64(time.Since(t).Nanoseconds()) / float64(time.Millisecond)
}

func (m *metricsImpl) SetScanDurationTime(start time.Time, scanner string, err error) {
	m.scanTimeDuration.With(prometheus.Labels{"Scanner": scanner, "Error": fmt.Sprintf("%t", err != nil)}).Observe(startTimeToMS(start))
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
		scanTimeDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: pkgMetrics.PrometheusNamespace,
			Subsystem: subsystem.String(),
			Name:      "scan_duration",
			Help:      "Amount of time it's taken to scan an image",
			Buckets:   prometheus.ExponentialBuckets(4, 2, 8),
		}, []string{"Scanner", "Error"}),
	}

	pkgMetrics.EmplaceCollector(
		m.metadataCacheHits,
		m.metadataCacheMisses,
		m.scanCacheHits,
		m.scanCacheMisses,
		m.scanTimeDuration,
	)

	return m
}
