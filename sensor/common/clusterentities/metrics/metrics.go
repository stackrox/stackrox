package metrics

import (
	"slices"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/pkg/metrics"
)

var (
	containersStored = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "num_containers_in_clusterentities_store",
		Help:      "A gauge to track the number of containers in the entity store",
	}, []string{"type"})

	ipsStored = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "num_ips_in_clusterentities_store",
		Help:      "A gauge to track the number of IPs in the entity store",
	}, []string{"type"})

	endpointsStored = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "num_endpoints_in_clusterentities_store",
		Help:      "A gauge to track the number of endpoints in the entity store",
	}, []string{"type"})

	// This metric is ideally always 0 - we do not expect one IP to have multiple owners,
	// but if that happens in the wild, we want to know
	ipsHavingMultipleContainers = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "ips_having_multiple_containers_total",
		Help:      "Count how many times a single IP was assigned to more than one container",
	}, []string{"ip", "containers"})
)

// UpdateNumberOfContainerIDs updates the metric tracking the number of containers stored in-memory store
func UpdateNumberOfContainerIDs(current, historical int) {
	containersStored.With(prometheus.Labels{"type": "current"}).Set(float64(current))
	containersStored.With(prometheus.Labels{"type": "historical"}).Set(float64(historical))
}

// UpdateNumberOfIPs updates the metric tracking the number of IPs stored in-memory store
func UpdateNumberOfIPs(current, historical int) {
	ipsStored.With(prometheus.Labels{"type": "current"}).Set(float64(current))
	ipsStored.With(prometheus.Labels{"type": "historical"}).Set(float64(historical))
}

// UpdateNumberOfEndpoints updates the metric tracking the number of endpoints stored in-memory store
func UpdateNumberOfEndpoints(current, historical int) {
	endpointsStored.With(prometheus.Labels{"type": "current"}).Set(float64(current))
	endpointsStored.With(prometheus.Labels{"type": "historical"}).Set(float64(historical))
}

// ObserveManyDeploymentsSharingSingleIP records a situation when one IP belongs to more than one container
func ObserveManyDeploymentsSharingSingleIP(ip string, containers []string) {
	// Prevent labels being doubled in form of:
	// metric{containers=[A,B]} 20, metric{containers=[B,A]} 5
	// instead of the expected:
	// metric{containers=[A,B]} 25
	slices.Sort(containers)
	ipsHavingMultipleContainers.With(
		prometheus.Labels{
			"ip":         ip,
			"containers": strings.Join(containers, ","),
		}).Inc()
}

func init() {
	prometheus.MustRegister(containersStored)
	prometheus.MustRegister(ipsStored)
	prometheus.MustRegister(endpointsStored)
	prometheus.MustRegister(ipsHavingMultipleContainers)
}
