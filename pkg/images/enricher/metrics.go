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
	IncrementEnricherSemaphoreQueueSize()
	DecrementEnricherSemaphoreQueueSize()
	IncrementEnricherSemaphoreHoldingSize()
	DecrementEnricherSemaphoreHoldingSize()
	SetScanDurationTime(start time.Time, scanner string, err error)
	SetImageVulnerabilityRetrievalTime(start time.Time, scanner string, err error)
}

type metricsImpl struct {
	metadataCacheHits   prometheus.Counter
	metadataCacheMisses prometheus.Counter

	scanTimeDuration           *prometheus.HistogramVec
	imageVulnRetrievalDuration *prometheus.HistogramVec

	enricherSemaphoreQueueSize   prometheus.Gauge
	enricherSemaphoreHoldingSize prometheus.Gauge
}

func (m *metricsImpl) IncrementMetadataCacheHit() {
	m.metadataCacheHits.Inc()
}

func (m *metricsImpl) IncrementMetadataCacheMiss() {
	m.metadataCacheMisses.Inc()
}

func (m *metricsImpl) IncrementEnricherSemaphoreQueueSize() {
	m.enricherSemaphoreQueueSize.Inc()
}

func (m *metricsImpl) DecrementEnricherSemaphoreQueueSize() {
	m.enricherSemaphoreQueueSize.Dec()
}

func (m *metricsImpl) IncrementEnricherSemaphoreHoldingSize() {
	m.enricherSemaphoreHoldingSize.Inc()
}

func (m *metricsImpl) DecrementEnricherSemaphoreHoldingSize() {
	m.enricherSemaphoreHoldingSize.Dec()
}

func startTimeToMS(t time.Time) float64 {
	return float64(time.Since(t).Nanoseconds()) / float64(time.Millisecond)
}

func (m *metricsImpl) SetScanDurationTime(start time.Time, scanner string, err error) {
	m.scanTimeDuration.With(prometheus.Labels{"Scanner": scanner, "Error": fmt.Sprintf("%t", err != nil)}).Observe(startTimeToMS(start))
}

func (m *metricsImpl) SetImageVulnerabilityRetrievalTime(start time.Time, scanner string, err error) {
	m.imageVulnRetrievalDuration.With(prometheus.Labels{"Scanner": scanner, "Error": fmt.Sprintf("%t", err != nil)}).Observe(startTimeToMS(start))
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
		scanTimeDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: pkgMetrics.PrometheusNamespace,
			Subsystem: subsystem.String(),
			Name:      "scan_duration",
			Help:      "Amount of time it's taken to scan an image in ms",
			Buckets:   prometheus.ExponentialBuckets(4, 2, 16),
		}, []string{"Scanner", "Error"}),
		imageVulnRetrievalDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: pkgMetrics.PrometheusNamespace,
			Subsystem: subsystem.String(),
			Name:      "image_vuln_retrieval_duration",
			Help:      "Amount of time it's taken to retrieve vulns for an image in ms",
			Buckets:   prometheus.ExponentialBuckets(4, 2, 16),
		}, []string{"Scanner", "Error"}),
		enricherSemaphoreQueueSize: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: pkgMetrics.PrometheusNamespace,
			Subsystem: subsystem.String(),
			Name:      "enricher_semaphore_queue_size",
			Help:      "A counter that tracks the size of the queues for the scan semaphores used in image scans.",
		}),
		enricherSemaphoreHoldingSize: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: pkgMetrics.PrometheusNamespace,
			Subsystem: subsystem.String(),
			Name:      "enricher_semaphore_holding_size",
			Help:      "A counter that tracks the number of requests successfully holding scanner semaphores",
		}),
	}

	pkgMetrics.EmplaceCollector(
		m.metadataCacheHits,
		m.metadataCacheMisses,
		m.scanTimeDuration,
		m.imageVulnRetrievalDuration,
		m.enricherSemaphoreQueueSize,
		m.enricherSemaphoreHoldingSize,
	)

	return m
}
