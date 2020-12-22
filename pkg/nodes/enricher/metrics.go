package enricher

import (
	"fmt"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	pkgMetrics "github.com/stackrox/rox/pkg/metrics"
)

// This interface encapsulates the metrics this package needs.
type metrics interface {
	IncrementScanCacheHit()

	IncrementScanCacheMiss()

	SetScanDurationTime(start time.Time, scanner string, err error)
}

type metricsImpl struct {
	scanCacheHits   prometheus.Counter
	scanCacheMisses prometheus.Counter

	scanTimeDuration *prometheus.HistogramVec
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
		scanCacheHits: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: pkgMetrics.PrometheusNamespace,
			Subsystem: subsystem.String(),
			Name:      "node_scan_cache_hits",
			Help:      "Number of cache hits in the node scan cache",
		}),
		scanCacheMisses: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: pkgMetrics.PrometheusNamespace,
			Subsystem: subsystem.String(),
			Name:      "node_scan_cache_misses",
			Help:      "Number of cache misses in the node scan cache",
		}),
		scanTimeDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: pkgMetrics.PrometheusNamespace,
			Subsystem: subsystem.String(),
			Name:      "node_scan_duration",
			Help:      "Amount of time it's taken to scan a node in ms",
			Buckets:   prometheus.ExponentialBuckets(4, 2, 16),
		}, []string{"Scanner", "Error"}),
	}

	pkgMetrics.EmplaceCollector(
		m.scanCacheHits,
		m.scanCacheMisses,
		m.scanTimeDuration,
	)

	return m
}
