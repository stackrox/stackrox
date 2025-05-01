package metrics

import (
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/pkg/metrics"
)

func init() {
	prometheus.MustRegister(
		// Host Connections
		NetworkConnectionInfoMessagesRcvd,
		NumUpdated,
		HostConnectionsOperations,
		IncomingConnectionsEndpoints,

		// Network Flows Manager
		FlowEnrichments,
		FlowEnrichmentEventsEndpoint,
		FlowEnrichmentEventsConnection,
		ExternalFlowCounter,
		InternalFlowCounter,
		activeFlowsCurrent,
		activeEndpointsCurrent,
		PurgerEvents,
		ActiveEndpointsPurgerDuration,
		NumUpdatedConnectionsEndpoints,

		// Other
		NetworkEntityFlowCounter, // flow directions and graph entities
		HostProcessesEvents,      // plop
		HostProcessesEnrichmentEvents,
	)
}

const (
	hostConnectionsPrefix = "host_connections_"
	netFlowManagerPrefix  = "network_flow_manager_"
)

// Metrics for network flows
var (
	// A networkConnectionInfo message arrives from collector

	// NetworkConnectionInfoMessagesRcvd - 1. Collector sends NetworkConnection Info messages where each contains endpoints and connections
	NetworkConnectionInfoMessagesRcvd = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      hostConnectionsPrefix + "msgs_received_per_node_total",
		Help:      "Total number of messages containing network flows received from Collector for a specific node",
	}, []string{"Hostname"})
	// NumUpdated - 2. Out of newly arrived endpoints and connections, only selected need an update
	NumUpdated = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      hostConnectionsPrefix + "num_updates",
		Help:      "Current number of network endpoints or connections being updated in the message from Collector received for a specific node",
	}, []string{"Hostname", "Type"})
	// HostConnectionsOperations - 3a. Out of the updates, only some result in adding the connection/endpoint to the map
	HostConnectionsOperations = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      hostConnectionsPrefix + "operations_total",
		Help:      "Total number of flows/endpoints added/removed in the host connections maps",
	}, []string{"op", "object"})
	// IncomingConnectionsEndpoints - 3b. how many Collector updates have the closeTS set and how many are unclosed
	// This is useful to investigate the behavior of Sensor with fake workloads when manipulating the `generateUnclosedEndpoints` param.
	IncomingConnectionsEndpoints = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      hostConnectionsPrefix + "incoming_objects_total",
		Help:      "Total number of incoming connections/endpoints received from Collector with their close TS set or unset",
	}, []string{"object", "closedTS"})
	// End of processing of the networkConnectionInfo message

	// FlowEnrichments - 4. All connections and endpoints kept in memory are enriched
	FlowEnrichments = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      netFlowManagerPrefix + "enrichments_total",
		Help: "Total number of enrichments started for a given object " +
			"(allows to calculate the percentage of events being enriched for " +
			"network_flow_manager_enrichment_endpoint_events_total and network_flow_manager_enrichment_connection_events_total)",
	}, []string{"object"})
	// FlowEnrichmentEventsEndpoint - 4a. Enrichment can have various outcomes. This metric stores the details about the outcomes for endpoints.
	FlowEnrichmentEventsEndpoint = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      netFlowManagerPrefix + "enrichment_endpoint_events_total",
		Help:      "Total number of events occurred to endpoints during the enrichment of network flows passed from collector",
	}, []string{"containerIDfound", "action", "isHistorical", "reason", "lastSeenSet", "rotten", "mature", "fresh"})
	// FlowEnrichmentEventsConnection - 4b. Enrichment can have various outcomes. This metric stores the details about the outcomes for connections.
	FlowEnrichmentEventsConnection = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      netFlowManagerPrefix + "enrichment_connection_events_total",
		Help:      "Total number of events occurred to connections during the enrichment of network flows passed from collector",
	}, []string{"containerIDfound", "action", "isHistorical", "reason", "lastSeenSet", "rotten", "mature", "fresh", "isExternal"})
	// ExternalFlowCounter - 4c. Counts the number of flows treated as external in the enrichment (will show edge to External Entities on the Network Graph).
	ExternalFlowCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      netFlowManagerPrefix + "external_flows",
		Help:      "Total number of external flows observed by Sensor enrichment",
	}, []string{"direction", "namespace"})
	// InternalFlowCounter - 4d. Counts the number of flows treated as internal in the enrichment (will show edge to Internal Entities on the Network Graph).
	InternalFlowCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      netFlowManagerPrefix + "internal_flows",
		Help:      "Total number of internal flows observed by Sensor enrichment",
	}, []string{"direction", "namespace"})

	// NumUpdatedConnectionsEndpoints - 5. An update is calculated between the states in consecutive enrichment ticks and the
	// difference is treated as new updates. That updates are sent to central.
	NumUpdatedConnectionsEndpoints = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      netFlowManagerPrefix + "num_sent_to_central_total",
		Help:      "A counter that tracks the total number of connections and endpoints being updated (i.e., sent to Central)",
	}, []string{"object"})
	activeFlowsCurrent = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      netFlowManagerPrefix + "active_network_flows_current",
		Help:      "A gauge that tracks the current active network flows in sensor",
	})
	activeEndpointsCurrent = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      netFlowManagerPrefix + "active_endpoints_current",
		Help:      "A gauge that tracks the current active endpoints in sensor",
	})
	PurgerEvents = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      netFlowManagerPrefix + "purger_events_total",
		Help:      "A counter that tracks the reasons for purging an object from memory",
	}, []string{"object", "purgeReason"})
	ActiveEndpointsPurgerDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      netFlowManagerPrefix + "purger_duration_seconds",
		Help:      "Time taken by a single purger run for all objects",
		Buckets:   []float64{.01, .05, .1, .25, .5, 1, 2.5, 5, 10},
	}, []string{"object"})

	NetworkEntityFlowCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      netFlowManagerPrefix + "entity_flows_total",
		Help:      "Total number of network entity flows observed by Sensor enrichment",
	}, []string{"direction", "namespace"})

	HostProcessesEvents = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      netFlowManagerPrefix + "processes_listening_on_port_events_total",
		Help:      "Total number of endpoints for processes listening on ports added/removed to hostConns",
	}, []string{"op"})
	HostProcessesEnrichmentEvents = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      netFlowManagerPrefix + "processes_listening_on_port_enrichment_events_total",
		Help:      "Total number of enrichment outcomes for the plop",
	}, []string{"containerIDfound", "action", "isHistorical", "reason", "lastSeenSet", "rotten", "mature", "fresh"})
)

func IncHostProcessesEnrichmentEvents(condIDfound, action, isHistorical string, reason string, lastSeenSet, rotten, mature, fresh bool) {
	HostProcessesEnrichmentEvents.With(prometheus.Labels{
		"containerIDfound": condIDfound,
		"action":           action,
		"isHistorical":     isHistorical,
		"reason":           reason,
		"lastSeenSet":      strconv.FormatBool(lastSeenSet),
		"rotten":           strconv.FormatBool(rotten),
		"mature":           strconv.FormatBool(mature),
		"fresh":            strconv.FormatBool(fresh)}).Inc()
}

func IncFlowEnrichmentEndpoint(condIDfound bool, action, isHistorical string, reason string, lastSeenSet, rotten, mature, fresh bool) {
	FlowEnrichmentEventsEndpoint.With(prometheus.Labels{
		"containerIDfound": strconv.FormatBool(condIDfound),
		"action":           action,
		"isHistorical":     isHistorical,
		"reason":           reason,
		"lastSeenSet":      strconv.FormatBool(lastSeenSet),
		"rotten":           strconv.FormatBool(rotten),
		"mature":           strconv.FormatBool(mature),
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
	activeFlowsCurrent.Set(float64(number))
}

// SetActiveEndpointsTotalGauge set the active endpoints total gauge.
func SetActiveEndpointsTotalGauge(number int) {
	activeEndpointsCurrent.Set(float64(number))
}
