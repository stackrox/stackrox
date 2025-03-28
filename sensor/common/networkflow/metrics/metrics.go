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
		NetworkConnectionInfoMessagesRcvd,
		FlowEnrichments,
		FlowEnrichmentEventsEndpoint,
		FlowEnrichmentEventsConnection,
		ExternalFlowCounter,
		InternalFlowCounter,
		NetworkEntityFlowCounter,
		HostConnections,
		HostProcessesRemoved,
		NumUpdated,
		activeFlowsTotal,
		activeEndpointsTotal,
		NumUpdatedConnectionsEndpoints,
	)
}

// Metrics for network flows
var (
	// A networkConnectionInfo message arrives from collector

	// NetworkConnectionInfoMessagesRcvd - 1. Collector sends NetworkConnection Info messages where each contains endpoints and connections
	NetworkConnectionInfoMessagesRcvd = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "network_connection_info_msgs_received_per_node_total",
		Help:      "Total number of messages containing network flows received from Collector for a specific node",
	}, []string{"Hostname"})
	// NumUpdated - 2. Out of newly arrived endpoints and connections, only selected need an update
	NumUpdated = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "network_connection_info_num_updates",
		Help:      "Current number of network endpoints or connections being updated in the message from Collector received for a specific node",
	}, []string{"Hostname", "Type"})
	// HostConnections - 3a. Out of the updates, only some result in adding dhe connection to the connections map
	HostConnections = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "network_connection_info_host_connections_total",
		Help:      "Total number of flows added/removed in the host connections maps",
	}, []string{"op"})
	// HostEndpoints - 3b. The same as 3a but for endpoints
	HostEndpoints = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "network_connection_info_host_endpoints_total",
		Help:      "Total number of endpoints added/removed in the host connections maps",
	}, []string{"op"})
	// End of processing of the networkConnectionInfo message

	NetworkFlowsPerNodeByType = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "network_flow_total_per_node_total",
		Help:      "Total number of network flows received for a specific node",
	}, []string{"Hostname", "Type", "Protocol"})
	ContainerEndpointsPerNode = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "network_endpoints_total_per_node_total",
		Help:      "Total number of container endpoint updates received for a specific node",
	}, []string{"Hostname", "Protocol"})

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
	}, []string{"containerIDfound", "action", "isHistorical", "reason", "lastSeenSet", "rotten", "mature", "fresh"})
	FlowEnrichmentEventsConnection = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "network_flow_enrichment_connection_events_total",
		Help:      "Total number of events occurred to connections during the enrichment of network flows passed from collector",
	}, []string{"containerIDfound", "action", "isHistorical", "reason", "lastSeenSet", "rotten", "mature", "fresh", "isExternal"})
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

	HostProcessesRemoved = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "processes_listening_on_port_removed_total",
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

func IncFlowEnrichmentConnection(condIDfound bool, action, isHistorical string, reason string, lastSeenSet, rotten, mature, fresh bool, isExternal string) {
	FlowEnrichmentEventsConnection.With(prometheus.Labels{
		"containerIDfound": strconv.FormatBool(condIDfound),
		"action":           action,
		"isHistorical":     isHistorical,
		"reason":           reason,
		"lastSeenSet":      strconv.FormatBool(lastSeenSet),
		"rotten":           strconv.FormatBool(rotten),
		"mature":           strconv.FormatBool(mature),
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
