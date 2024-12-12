package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/pkg/metrics"
)

func init() {
	prometheus.MustRegister(
		ClusterLocalHostsCount,
		GlobalSecretEntriesCount,
		PullSecretEntriesCount,
		PullSecretEntriesSize,
		CentralIntegrationsCount,
		TLSCheckCount,
		TLSCheckDuration,
	)
}

// Registry store metrics.
var (
	ClusterLocalHostsCount = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "registry_store_cluster_local_hosts_count",
		Help:      "Current number of cluster local (i.e. OCP Internal Registry) hosts inserted into the registry store",
	})

	GlobalSecretEntriesCount = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "registry_store_global_secret_entries_count",
		Help:      "Current number of stored global registry entries",
	})

	PullSecretEntriesCount = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "registry_store_pull_secret_entries_count",
		Help:      "Current number of stored pull secret entries",
	})

	PullSecretEntriesSize = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "registry_store_pull_secret_entries_size",
		Help:      "Rough size in bytes of the currently stored pull secret entries",
	})

	CentralIntegrationsCount = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "registry_store_central_integrations_count",
		Help:      "Current number of stored image integrations from Central",
	})

	TLSCheckCount = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "registry_store_tls_check_count",
		Help:      "The total number of TLS checks requested via the registry store",
	})

	TLSCheckDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "registry_store_tls_check_duration_seconds",
		Help:      "Time taken in seconds to perform a TLS check on a registry host (does not include TLS check cache hits)",
	})
)

// SetClusterLocalHostsCount updates the count of stored cluster local registry hosts.
func SetClusterLocalHostsCount(count int) {
	ClusterLocalHostsCount.Set(float64(count))
}

// SetGlobalSecretEntriesCount updates the count of global secret entries in the registry store.
func SetGlobalSecretEntriesCount(count int) {
	GlobalSecretEntriesCount.Set(float64(count))
}

// IncrementPullSecretEntriesCount adds to the count of pull secret entries in the registry store.
func IncrementPullSecretEntriesCount(value int) {
	PullSecretEntriesCount.Add(float64(value))
}

// DecrementPullSecretEntriesCount subtracts from the count of pull secrets in the registry store.
func DecrementPullSecretEntriesCount(value int) {
	PullSecretEntriesCount.Sub(float64(value))
}

// IncrementPullSecretEntriesSize adds to the total size of pull secret entries in the registry store.
func IncrementPullSecretEntriesSize(value int) {
	PullSecretEntriesSize.Add(float64(value))
}

// DecrementPullSecretEntriesSize subtracts from the total size of pull secret entries in the registry store.
func DecrementPullSecretEntriesSize(value int) {
	PullSecretEntriesSize.Sub(float64(value))
}

// SetCentralIntegrationCount updates the count of image integrations from Central in the registry store.
func SetCentralIntegrationCount(value int) {
	CentralIntegrationsCount.Set(float64(value))
}

// IncrementTLSCheckCount adds to the total count of TLS check requests made via the registry store.
func IncrementTLSCheckCount() {
	TLSCheckCount.Inc()
}

// ObserveTLSCheckDuration observes the time in seconds taken to perform a TLS check.
func ObserveTLSCheckDuration(t time.Duration) {
	TLSCheckDuration.Observe(t.Seconds())
}

// ResetRegistryMetrics resets the count and size metrics for registry store entries.
func ResetRegistryMetrics() {
	ClusterLocalHostsCount.Set(0)
	GlobalSecretEntriesCount.Set(0)
	PullSecretEntriesCount.Set(0)
	PullSecretEntriesSize.Set(0)
	CentralIntegrationsCount.Set(0)
}
