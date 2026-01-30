package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type metricsRegistry struct {
	requests *prometheus.CounterVec
	bytes    prometheus.Counter
	latency  prometheus.Histogram

	// Runtime distribution histograms (observed per report)
	reportedPackages prometheus.Histogram
	reportedInterval prometheus.Histogram

	// Distribution stats gauges
	packagesMean prometheus.Gauge
	packagesP50  prometheus.Gauge
	packagesP95  prometheus.Gauge
	packagesP99  prometheus.Gauge
	intervalMean prometheus.Gauge
	intervalP50  prometheus.Gauge
	intervalP95  prometheus.Gauge
	intervalP99  prometheus.Gauge
}

func newMetricsRegistry() *metricsRegistry {
	m := &metricsRegistry{
		requests: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "vsock_loadgen_requests_total",
				Help: "Total vsock load generator requests by result.",
			},
			[]string{"result"},
		),
		bytes: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "vsock_loadgen_bytes_total",
				Help: "Total bytes sent to the relay.",
			},
		),
		reportedPackages: prometheus.NewHistogram(
			prometheus.HistogramOpts{
				Name:    "vsock_loadgen_packages_reported",
				Help:    "Observed package counts in index reports sent by the load generator.",
				Buckets: []float64{1, 5, 10, 25, 50, 75, 100, 150, 200, 300, 400, 600, 800, 1000},
			},
		),
		reportedInterval: prometheus.NewHistogram(
			prometheus.HistogramOpts{
				Name:    "vsock_loadgen_report_interval_seconds",
				Help:    "Observed interval in seconds between VM index reports (including jitter).",
				Buckets: []float64{0.5, 1, 2, 3, 5, 8, 10, 15, 20, 30, 45, 60, 90, 120},
			},
		),
		latency: prometheus.NewHistogram(
			prometheus.HistogramOpts{
				Name:    "vsock_loadgen_request_latency_seconds",
				Help:    "Request latency in seconds.",
				Buckets: prometheus.ExponentialBuckets(0.01, 1.4, 15),
			},
		),
		packagesMean: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "vsock_loadgen_packages_mean",
				Help: "Mean package count across all VMs.",
			},
		),
		packagesP50: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "vsock_loadgen_packages_p50",
				Help: "P50 (median) package count across all VMs.",
			},
		),
		packagesP95: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "vsock_loadgen_packages_p95",
				Help: "P95 package count across all VMs.",
			},
		),
		packagesP99: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "vsock_loadgen_packages_p99",
				Help: "P99 package count across all VMs.",
			},
		),
		intervalMean: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "vsock_loadgen_interval_mean_seconds",
				Help: "Mean report interval in seconds across all VMs.",
			},
		),
		intervalP50: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "vsock_loadgen_interval_p50_seconds",
				Help: "P50 (median) report interval in seconds across all VMs.",
			},
		),
		intervalP95: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "vsock_loadgen_interval_p95_seconds",
				Help: "P95 report interval in seconds across all VMs.",
			},
		),
		intervalP99: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "vsock_loadgen_interval_p99_seconds",
				Help: "P99 report interval in seconds across all VMs.",
			},
		),
	}
	prometheus.MustRegister(m.requests, m.bytes, m.latency,
		m.reportedPackages, m.reportedInterval,
		m.packagesMean, m.packagesP50, m.packagesP95, m.packagesP99,
		m.intervalMean, m.intervalP50, m.intervalP95, m.intervalP99)
	return m
}

func (m *metricsRegistry) observeSuccess(latency time.Duration, bytes int) {
	m.requests.WithLabelValues("success").Inc()
	m.bytes.Add(float64(bytes))
	m.latency.Observe(latency.Seconds())
}

func (m *metricsRegistry) observeFailure(reason string) {
	m.requests.WithLabelValues(reason).Inc()
}

// observeReport tracks per-report observed attributes (package count and interval).
func (m *metricsRegistry) observeReport(packages int, interval time.Duration) {
	if packages > 0 {
		m.reportedPackages.Observe(float64(packages))
	}
	if interval > 0 {
		m.reportedInterval.Observe(interval.Seconds())
	}
}

// computeDistributionStats computes statistics from VM configurations and sets Prometheus gauges.
func computeDistributionStats(vmConfigs []vmConfig, metrics *metricsRegistry) {
	if len(vmConfigs) == 0 {
		return
	}

	// Extract package counts and intervals
	packages := make([]int, len(vmConfigs))
	intervals := make([]float64, len(vmConfigs))
	var sumPackages int
	var sumIntervals float64

	for i, vmCfg := range vmConfigs {
		packages[i] = vmCfg.numPackages
		intervals[i] = vmCfg.reportInterval.Seconds()
		sumPackages += vmCfg.numPackages
		sumIntervals += intervals[i]
	}

	// Sort for percentile calculation
	sort.Ints(packages)
	sort.Float64s(intervals)

	// Compute statistics
	meanPackages := float64(sumPackages) / float64(len(vmConfigs))
	meanIntervals := sumIntervals / float64(len(vmConfigs))

	p50Packages := percentileInt(packages, 50)
	p95Packages := percentileInt(packages, 95)
	p99Packages := percentileInt(packages, 99)

	p50Intervals := percentileFloat64(intervals, 50)
	p95Intervals := percentileFloat64(intervals, 95)
	p99Intervals := percentileFloat64(intervals, 99)

	// Set Prometheus gauges
	metrics.packagesMean.Set(meanPackages)
	metrics.packagesP50.Set(float64(p50Packages))
	metrics.packagesP95.Set(float64(p95Packages))
	metrics.packagesP99.Set(float64(p99Packages))

	metrics.intervalMean.Set(meanIntervals)
	metrics.intervalP50.Set(p50Intervals)
	metrics.intervalP95.Set(p95Intervals)
	metrics.intervalP99.Set(p99Intervals)

	// Log summary
	log.Infof("Distribution stats: packages mean=%.2f p50=%d p95=%d p99=%d, intervals mean=%.2fs p50=%.2fs p95=%.2fs p99=%.2fs",
		meanPackages, p50Packages, p95Packages, p99Packages,
		meanIntervals, p50Intervals, p95Intervals, p99Intervals)
}

// percentileInt calculates the percentile value from a sorted slice of integers.
func percentileInt(sorted []int, p float64) int {
	if len(sorted) == 0 {
		return 0
	}
	index := (p / 100.0) * float64(len(sorted)-1)
	lower := int(index)
	upper := lower + 1
	if upper >= len(sorted) {
		return sorted[len(sorted)-1]
	}
	weight := index - float64(lower)
	return int(float64(sorted[lower])*(1-weight) + float64(sorted[upper])*weight)
}

// percentileFloat64 calculates the percentile value from a sorted slice of float64s.
func percentileFloat64(sorted []float64, p float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	index := (p / 100.0) * float64(len(sorted)-1)
	lower := int(index)
	upper := lower + 1
	if upper >= len(sorted) {
		return sorted[len(sorted)-1]
	}
	weight := index - float64(lower)
	return sorted[lower]*(1-weight) + sorted[upper]*weight
}

func serveMetrics(ctx context.Context, port int) {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()

	log.Infof("metrics server listening on :%d", port)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Errorf("metrics server error: %v", err)
	}
}
