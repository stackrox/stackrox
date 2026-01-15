package watcher

import (
	"fmt"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/pkg/metrics"
)

func init() {
	prometheus.MustRegister(
		pollDurationHistogram,
		repositoryCountGauge,
		tagsListedGauge,
		metadataFetchErrors,
		scanDuration,
	)
}

var (
	pollDurationHistogram = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: metrics.PrometheusNamespace,
			Subsystem: "base_image_watcher",
			Name:      "poll_duration_seconds",
			Help:      "Time taken to complete a poll cycle",
			Buckets:   prometheus.ExponentialBuckets(0.1, 2, 10), // 0.1s to ~102s
		},
		[]string{"error"},
	)

	repositoryCountGauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: "base_image_watcher",
		Name:      "repositories_total",
		Help:      "Number of base image repositories being watched",
	})

	tagsListedGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: "base_image_watcher",
		Name:      "tags_listed",
		Help:      "Current number of tags listed for each repository",
	}, []string{
		"registry_domain",
		"repository_path",
		"source",
	})

	metadataFetchErrors = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: "base_image_watcher",
		Name:      "metadata_fetch_errors",
		Help:      "Number of metadata fetch errors per repository",
	}, []string{
		"registry_domain",
		"repository_path",
		"source",
	})

	scanDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: "base_image_watcher",
		Name:      "scan_duration_seconds",
		Help:      "Time taken to scan a repository (list tags + fetch metadata)",
		Buckets:   prometheus.ExponentialBuckets(0.1, 2, 12), // 0.1s to ~409s
	}, []string{
		"registry_domain",
		"repository_path",
		"source",
		"error",
	})
)

func recordPollDuration(seconds float64, err error) {
	pollDurationHistogram.WithLabelValues(fmt.Sprintf("%t", err != nil)).Observe(seconds)
}

func recordRepositoryCount(count int) {
	repositoryCountGauge.Set(float64(count))
}

func recordScanDuration(registryDomain, repositoryPath, source string, startTime time.Time, metadataCount int, errorCount int, err error) {
	duration := time.Since(startTime).Seconds()
	scanDuration.With(prometheus.Labels{
		"registry_domain": registryDomain,
		"repository_path": repositoryPath,
		"source":          source,
		"error":           fmt.Sprintf("%t", err != nil),
	}).Observe(duration)
	if err == nil {
		tagsListedGauge.With(prometheus.Labels{
			"registry_domain": registryDomain,
			"repository_path": repositoryPath,
			"source":          source,
		}).Set(float64(metadataCount))
		metadataFetchErrors.With(prometheus.Labels{
			"registry_domain": registryDomain,
			"repository_path": repositoryPath,
			"source":          source,
		}).Set(float64(errorCount))
	}
}
