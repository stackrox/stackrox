package metrics

import (
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
		Help:      "Current number of stored image integrations from Central.",
	})
)

// IncrementClusterLocalHostsCount adds to the count of stored cluster local registry hosts.
func IncrementClusterLocalHostsCount() {
	ClusterLocalHostsCount.Add(1)
}

// SetGlobalSecretEntriesCount updates the count of global secret entries in the registry store.
func SetGlobalSecretEntriesCount(size int) {
	GlobalSecretEntriesCount.Set(float64(size))
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
func SetCentralIntegrationCount(size int) {
	CentralIntegrationsCount.Set(float64(size))
}

// ResetRegistryStoreMetrics resets the counts of all registry store metrics.
func ResetRegistryStoreMetrics() {
	ClusterLocalHostsCount.Set(0)
	GlobalSecretEntriesCount.Set(0)
	PullSecretEntriesCount.Set(0)
	PullSecretEntriesSize.Set(0)
	CentralIntegrationsCount.Set(0)
}
