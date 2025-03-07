package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/pkg/metrics"
)

func init() {
	prometheus.MustRegister(
		NetworkFlowsPerNodeByType,
		ContainerEndpointsPerNode,
		NetworkFlowMessagesPerNode,
		ContainerIDMisses,
		ExternalFlowCounter,
		InternalFlowCounter,
		NetworkEntityFlowCounter,
		HostConnectionsAdded,
		HostConnectionsRemoved,
		HostEndpointsAdded,
		HostEndpointsRemoved,
		activeFlowsTotal,
		activeEndpointsTotal,
	)
}

// Metrics for network flows
var (
	NetworkFlowsPerNodeByType = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "network_flow_total_per_node",
		Help:      "Total number of network flows received for a specific node",
	}, []string{"Hostname", "Type", "Protocol"})
	ContainerEndpointsPerNode = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "network_endpoints_total_per_node",
		Help:      "Total number of container endpoint updates received for a specific node",
	}, []string{"Hostname", "Protocol"})
	NetworkFlowMessagesPerNode = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "network_flow_msgs_received_per_node",
		Help:      "Total number of network flows received for a specific node",
	}, []string{"Hostname"})
	ContainerIDMisses = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "network_flow_misses_container_lookup",
		Help:      "Total number of misses on container lookup for network flows",
	}, []string{"status"})
	ExternalFlowCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "network_flow_external_flows",
		Help:      "Total number of external flows observed by Sensor enrichment",
	}, []string{"direction", "namespace"})
	InternalFlowCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "network_flow_internal_flows",
		Help:      "Total number of internal flows observed by Sensor enrichment",
	}, []string{"direction", "namespace"})
	NetworkEntityFlowCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "network_flow_entity_flows",
		Help:      "Total number of network entity flows observed by Sensor enrichment",
	}, []string{"direction", "namespace"})
	HostConnectionsAdded = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "network_flow_host_connections_added",
		Help:      "Total number of flows stored in the host connections maps",
	})
	HostConnectionsRemoved = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "network_flow_host_connections_removed",
		Help:      "Total number of flows stored in the host connections maps",
	})
	HostEndpointsAdded = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "network_flow_host_endpoints_added",
		Help:      "Total number of endpoints stored in the host endpoints maps",
	})
	HostEndpointsRemoved = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "network_flow_host_endpoints_removed",
		Help:      "Total number of endpoints stored in the host connections maps",
	})
	HostProcessesRemoved = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "processes_listening_on_port_removed",
		Help:      "Total number of processes listening on ports",
	})
	activeFlowsTotal = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "active_network_flows_total",
		Help:      "A gauge that tracks the total active network flows in sensor",
	})
	activeEndpointsTotal = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "active_endpoints_total",
		Help:      "A gauge that tracks the total active endpoints in sensor",
	})
)

// SetActiveFlowsTotalGauge set the active network flows total gauge.
func SetActiveFlowsTotalGauge(number int) {
	activeFlowsTotal.Set(float64(number))
}

// SetActiveEndpointsTotalGauge set the active endpoints total gauge.
func SetActiveEndpointsTotalGauge(number int) {
	activeEndpointsTotal.Set(float64(number))
}
