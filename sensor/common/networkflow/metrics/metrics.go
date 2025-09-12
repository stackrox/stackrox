package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/pkg/metrics"
)

func init() {
	prometheus.MustRegister(
		EnrichmentCollectionsSize,
		EnrichmentCollectionsSizeBytes,

		// Host Connections
		NetworkConnectionInfoMessagesRcvd,
		IncomingConnectionsEndpointsGauge,
		HostConnectionsOperations,
		IncomingConnectionsEndpointsCounter,

		// Network Flows Manager
		FlowEnrichmentEventsEndpoint,
		FlowEnrichmentEventsConnection,
		ExternalFlowCounter,
		InternalFlowCounter,
		activeFlowsCurrent,
		activeEndpointsCurrent,
		PurgerEvents,
		PurgerRunDuration,
		NumUpdatesSentToCentralCounter,
		NumUpdatesSentToCentralGauge,

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
	EnrichmentCollectionsSize = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      hostConnectionsPrefix + "collections_size_current",
		Help:      "Current size (number of elements) of given collection involved in enrichment",
	}, []string{"uc", "Name", "Type"})
	EnrichmentCollectionsSizeBytes = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      hostConnectionsPrefix + "collections_size_current_bytes",
		Help:      "Current size in bytes of given collection involved in enrichment",
	}, []string{"uc", "Name", "Type"})
	// A networkConnectionInfo message arrives from collector

	// NetworkConnectionInfoMessagesRcvd - 1. Collector sends NetworkConnection Info messages where each contains endpoints and connections
	NetworkConnectionInfoMessagesRcvd = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      hostConnectionsPrefix + "msgs_received_per_node_total",
		Help:      "Total number of messages containing network flows received from Collector for a specific node",
	}, []string{"Hostname"})
	// IncomingConnectionsEndpointsGauge - 2. Out of newly arrived endpoints and connections, only selected need an update
	IncomingConnectionsEndpointsGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      hostConnectionsPrefix + "incoming_objects_current",
		Help:      "Current number of network endpoints or connections being updated in the message from Collector received for a specific node",
	}, []string{"Hostname", "Type", "status"})
	// HostConnectionsOperations - 3a. Out of the updates, only some result in adding the connection/endpoint to the map
	HostConnectionsOperations = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      hostConnectionsPrefix + "operations_total",
		Help:      "Total number of flows/endpoints added/removed in the host connections maps",
	}, []string{"op", "object"})
	// IncomingConnectionsEndpointsCounter - 3b. how many Collector updates have the closeTS set and how many are unclosed
	// This is useful to investigate the behavior of Sensor with fake workloads when manipulating the `generateUnclosedEndpoints` param.
	IncomingConnectionsEndpointsCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      hostConnectionsPrefix + "incoming_objects_total",
		Help:      "Total number of incoming connections/endpoints received from Collector with their close status",
	}, []string{"object", "status"})
	// End of processing of the networkConnectionInfo message

	// FlowEnrichmentEventsEndpoint - 4a. Enrichment can have various outcomes. This metric stores the details about the outcomes for endpoints.
	FlowEnrichmentEventsEndpoint = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      netFlowManagerPrefix + "enrichment_endpoint_events_total",
		Help:      "Total number of events occurred to endpoints during the enrichment of network flows passed from collector",
	}, []string{"containerIDfound", "result", "action", "isHistorical", "reason", "isClosed", "rotten", "mature", "fresh"})
	// FlowEnrichmentEventsConnection - 4b. Enrichment can have various outcomes. This metric stores the details about the outcomes for connections.
	FlowEnrichmentEventsConnection = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      netFlowManagerPrefix + "enrichment_connection_events_total",
		Help:      "Total number of events occurred to connections during the enrichment of network flows passed from collector",
	}, []string{"containerIDfound", "result", "action", "isHistorical", "reason", "isClosed", "rotten", "mature", "fresh", "isExternal"})
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

	// NumUpdatesSentToCentralCounter - 5. An update is calculated between the states in consecutive enrichment ticks and the
	// difference is treated as new updates. That updates are sent to central.
	NumUpdatesSentToCentralCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      netFlowManagerPrefix + "num_sent_to_central_total",
		Help:      "A counter that tracks the total number of connections and endpoints being updated (i.e., sent to Central)",
	}, []string{"object"})
	NumUpdatesSentToCentralGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      netFlowManagerPrefix + "num_sent_to_central_current",
		Help:      "A gauge that tracks the current number of connections and endpoints being updated (i.e., sent to Central)",
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
	PurgerRunDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
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
	}, []string{"containerIDfound", "result", "action", "isHistorical", "reason", "isClosed", "rotten", "mature", "fresh"})
)

// SetActiveFlowsTotalGauge set the active network flows total gauge.
func SetActiveFlowsTotalGauge(number int) {
	activeFlowsCurrent.Set(float64(number))
}

// SetActiveEndpointsTotalGauge set the active endpoints total gauge.
func SetActiveEndpointsTotalGauge(number int) {
	activeEndpointsCurrent.Set(float64(number))
}
