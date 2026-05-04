package datastore

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/pkg/metrics"
)

var (
	watcherUpsertTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "key_bundle_watcher_upsert_total",
		Help:      "Total number of key bundle upsert attempts by the watcher, labeled by result",
	}, []string{"result"})

	watcherFileErrorTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "key_bundle_watcher_file_error_total",
		Help:      "Total number of file-level errors in the watcher (stat, read, oversize, parse failures)",
	})

	watcherKeyCount = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "key_bundle_watcher_key_count",
		Help:      "Number of keys in the most recently applied Red Hat signing key bundle",
	})

	watcherLastSuccessTimestamp = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "key_bundle_watcher_last_success_timestamp",
		Help:      "Unix timestamp of the last successful key bundle upsert by the watcher",
	})

	updaterDownloadTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "key_bundle_updater_download_total",
		Help:      "Total number of key bundle download attempts by the updater, labeled by result",
	}, []string{"result"})

	updaterLastSuccessTimestamp = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "key_bundle_updater_last_success_timestamp",
		Help:      "Unix timestamp of the last successful key bundle download by the updater",
	})

	updaterDownloadDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "key_bundle_updater_download_duration_seconds",
		Help:      "Duration of key bundle download attempts by the updater",
		Buckets:   prometheus.DefBuckets,
	})
)

func init() {
	metrics.EmplaceCollector(
		watcherUpsertTotal,
		watcherFileErrorTotal,
		watcherKeyCount,
		watcherLastSuccessTimestamp,
		updaterDownloadTotal,
		updaterLastSuccessTimestamp,
		updaterDownloadDuration,
	)
}
