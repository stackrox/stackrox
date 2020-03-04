package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/pkg/metrics"
)

func init() {
	prometheus.MustRegister(NetworkFlowsPerNodeByType)
	prometheus.MustRegister(NetworkFlowMessagesPerNode)
	prometheus.MustRegister(ContainerIDMisses)
	prometheus.MustRegister(ExternalFlowCounter)
	prometheus.MustRegister(HostConnectionsRemoved)
	prometheus.MustRegister(HostConnectionsAdded)
}

// Metrics for network flows
var (
	NetworkFlowsPerNodeByType = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "network_flow_total_per_node",
		Help:      "Total number of network flows received for a specific node",
	}, []string{"Hostname", "Type", "Protocol"})
	NetworkFlowMessagesPerNode = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "network_flow_msgs_received_per_node",
		Help:      "Total number of network flows received for a specific node",
	}, []string{"Hostname"})
	ContainerIDMisses = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "network_flow_misses_container_lookup",
		Help:      "Total number of misses on container lookup for network flows",
	})
	ExternalFlowCounter = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "network_flow_external_flows",
		Help:      "Total number of external flows",
	})

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
)
