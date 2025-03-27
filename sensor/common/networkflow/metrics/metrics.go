package metrics

import (
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/pkg/metrics"
)

func init() {
	prometheus.MustRegister(
		NetworkFlowsPerNodeByType,
		ContainerEndpointsPerNode,
		NetworkFlowMessagesPerNode,
		FlowEnrichments,
		FlowEnrichmentEventsEndpoint,
		FlowEnrichmentEventsConnection,
		ExternalFlowCounter,
		InternalFlowCounter,
		NetworkEntityFlowCounter,
		HostConnectionsAdded,
		HostConnectionsRemoved,
		HostEndpointsAdded,
		HostEndpointsRemoved,
		activeFlowsTotal,
		activeEndpointsTotal,
		NumUpdatedConnectionsEndpoints,
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
	FlowEnrichments = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "network_flow_enrichments_total",
		Help: "Total number of enrichments started for a given object " +
			"(allows to calculate the percentage of events being enriched for " +
			"network_flow_enrichment_endpoint_events_total and network_flow_enrichment_connection_events_total)",
	}, []string{"object"})
	FlowEnrichmentEventsEndpoint = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "network_flow_enrichment_endpoint_events_total",
		Help:      "Total number of events occurred to endpoints during the enrichment of network flows passed from collector",
	}, []string{"containerIDfound", "action", "isHistorical", "reason", "lastSeenSet", "rotten", "expired", "fresh"})
	FlowEnrichmentEventsConnection = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "network_flow_enrichment_connection_events_total",
		Help:      "Total number of events occurred to connections during the enrichment of network flows passed from collector",
	}, []string{"containerIDfound", "action", "isHistorical", "reason", "lastSeenSet", "rotten", "expired", "fresh", "isExternal"})
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
	NumUpdatedConnectionsEndpoints = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "num_updated",
		Help:      "A gauge that tracks the number of connections and endpoints being updated (i.e., sent to Central) in a given tick",
	}, []string{"object"})
	activeFlowsTotal = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "active_network_flows_current",
		Help:      "A gauge that tracks the current  active network flows in sensor",
	})
	activeEndpointsTotal = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "active_endpoints_current",
		Help:      "A gauge that tracks the current active endpoints in sensor",
	})
)

func IncFlowEnrichmentEndpoint(condIDfound bool, action, isHistorical string, reason string, lastSeenSet, rotten, expired, fresh bool) {
	FlowEnrichmentEventsEndpoint.With(prometheus.Labels{
		"containerIDfound": strconv.FormatBool(condIDfound),
		"action":           action,
		"isHistorical":     isHistorical,
		"reason":           reason,
		"lastSeenSet":      strconv.FormatBool(lastSeenSet),
		"rotten":           strconv.FormatBool(rotten),
		"expired":          strconv.FormatBool(expired),
		"fresh":            strconv.FormatBool(fresh)}).Inc()
}

func IncFlowEnrichmentConnection(condIDfound bool, action, isHistorical string, reason string, lastSeenSet, rotten, expired, fresh bool, isExternal string) {
	FlowEnrichmentEventsConnection.With(prometheus.Labels{
		"containerIDfound": strconv.FormatBool(condIDfound),
		"action":           action,
		"isHistorical":     isHistorical,
		"reason":           reason,
		"lastSeenSet":      strconv.FormatBool(lastSeenSet),
		"rotten":           strconv.FormatBool(rotten),
		"expired":          strconv.FormatBool(expired),
		"fresh":            strconv.FormatBool(fresh),
		"isExternal":       isExternal}).Inc()
}

// SetActiveFlowsTotalGauge set the active network flows total gauge.
func SetActiveFlowsTotalGauge(number int) {
	activeFlowsTotal.Set(float64(number))
}

// SetActiveEndpointsTotalGauge set the active endpoints total gauge.
func SetActiveEndpointsTotalGauge(number int) {
	activeEndpointsTotal.Set(float64(number))
}
