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
		tagListingDuration,
		tagsListedGauge,
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

	tagListingDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: "base_image_watcher",
		Name:      "tag_listing_duration_seconds",
		Help:      "Time taken to list and filter tags from registry",
		Buckets:   prometheus.ExponentialBuckets(0.01, 2, 14), // 10ms to ~163s
	}, []string{
		"registry_domain",
		"repository_path",
		"error",
	})

	tagsListedGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: "base_image_watcher",
		Name:      "tags_listed",
		Help:      "Current number of tags listed for each repository",
	}, []string{
		"registry_domain",
		"repository_path",
	})
)

func recordPollDuration(seconds float64, err error) {
	pollDurationHistogram.WithLabelValues(fmt.Sprintf("%t", err != nil)).Observe(seconds)
}

func recordRepositoryCount(count int) {
	repositoryCountGauge.Set(float64(count))
}

func recordTagListDuration(registryDomain, repositoryPath string, startTime time.Time, tagCount int, err error) {
	duration := time.Since(startTime).Seconds()
	tagListingDuration.With(prometheus.Labels{
		"registry_domain": registryDomain,
		"repository_path": repositoryPath,
		"error":           fmt.Sprintf("%t", err != nil),
	}).Observe(duration)
	if err == nil {
		tagsListedGauge.With(prometheus.Labels{
			"registry_domain": registryDomain,
			"repository_path": repositoryPath,
		}).Set(float64(tagCount))
	}
}
