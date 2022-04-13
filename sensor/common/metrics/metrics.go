package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/stackrox/generated/internalapi/central"
	"github.com/stackrox/stackrox/pkg/metrics"
)

var (
	// Panics encountered
	panicCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "panic_counter",
		Help:      "Number of panic calls within Sensor.",
	}, []string{"FunctionName"})

	processDedupeCacheHits = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "dedupe_cache_hits",
		Help:      "A counter of the total number of times we've deduped the process passed in",
	})

	processDedupeCacheMisses = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "dedupe_cache_misses",
		Help:      "A counter of the total number of times we've passed through the dedupe cache",
	})

	processEnrichmentDrops = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "process_enrichment_drops",
		Help:      "A counter of the total number of times we've dropped enriching process indicators",
	})

	processEnrichmentHits = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "process_enrichment_hits",
		Help:      "A counter of the total number of times we've successfully enriched process indicators",
	})

	processEnrichmentLRUCacheSize = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "process_enrichment_cache_size",
		Help:      "A gauge to track the enrichment lru cache size",
	})

	sensorIndicatorChannelFullCounter = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "indicators_channel_indicator_dropped_counter",
		Help:      "A counter of the total number of times we've dropped indicators from the indicators channel because it was full",
	})

	totalNetworkFlowsSentCounter = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "total_network_flows_sent_counter",
		Help:      "A counter of the total number of network flows sent to Central by Sensor",
	})

	totalNetworkFlowsReceivedCounter = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "total_network_flows_sensor_received_counter",
		Help:      "A counter of the total number of network flows received by Sensor from Collector",
	})

	totalNetworkEndpointsSentCounter = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "total_network_endpoints_sent_counter",
		Help:      "A counter of the total number of network endpoints sent to Central by Sensor",
	})

	totalNetworkEndpointsReceivedCounter = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "total_network_endpoints_received_counter",
		Help:      "A counter of the total number of network endpoints received by Sensor from Collector",
	})

	sensorEvents = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "sensor_events",
		Help:      "A counter for the total number of events sent from Sensor to Central",
	}, []string{"Action", "ResourceType", "Type"})

	k8sObjectCounts = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "k8s_events",
		Help:      "A counter for the total number of typed k8s events processed by Sensor",
	}, []string{"Action", "Resource"})

	k8sObjectIngestionToSendDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "k8s_event_ingestion_to_send_duration",
		Help:      "Time taken to fully process an event from Kubernetes",
		Buckets:   prometheus.ExponentialBuckets(4, 2, 8),
	}, []string{"Action", "Resource", "Dispatcher", "Type"})

	k8sObjectProcessingDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "k8s_event_processing_duration",
		Help:      "Time taken to fully process an event from Kubernetes",
		Buckets:   prometheus.ExponentialBuckets(4, 2, 8),
	}, []string{"Action", "Resource", "Dispatcher"})
)

// IncrementPanicCounter increments the number of panic calls seen in a function
func IncrementPanicCounter(functionName string) {
	panicCounter.With(prometheus.Labels{"FunctionName": functionName}).Inc()
}

// IncrementProcessDedupeCacheHits increments the number of times we deduped a process
func IncrementProcessDedupeCacheHits() {
	processDedupeCacheHits.Inc()
}

// IncrementProcessDedupeCacheMisses increments the number of times we failed to dedupe a process
func IncrementProcessDedupeCacheMisses() {
	processDedupeCacheMisses.Inc()
}

// RegisterSensorIndicatorChannelFullCounter registers the number of indicators dropped
func RegisterSensorIndicatorChannelFullCounter() {
	sensorIndicatorChannelFullCounter.Inc()
}

// IncrementTotalNetworkFlowsSentCounter registers the total number of flows processed
func IncrementTotalNetworkFlowsSentCounter(numberOfFlows int) {
	totalNetworkFlowsSentCounter.Add(float64(numberOfFlows))
}

// IncrementTotalNetworkFlowsReceivedCounter registers the total number of flows received
func IncrementTotalNetworkFlowsReceivedCounter(numberOfFlows int) {
	totalNetworkFlowsReceivedCounter.Add(float64(numberOfFlows))
}

// IncrementTotalNetworkEndpointsSentCounter increments the total number of endpoints sent
func IncrementTotalNetworkEndpointsSentCounter(numberOfEndpoints int) {
	totalNetworkEndpointsSentCounter.Add(float64(numberOfEndpoints))
}

// IncrementTotalNetworkEndpointsReceivedCounter increments the total number of endpoints received
func IncrementTotalNetworkEndpointsReceivedCounter(numberOfEndpoints int) {
	totalNetworkEndpointsReceivedCounter.Add(float64(numberOfEndpoints))
}

// IncrementProcessEnrichmentDrops increments the number of times we could not enrich.
func IncrementProcessEnrichmentDrops() {
	processEnrichmentDrops.Inc()
}

// IncrementProcessEnrichmentHits increments the number of times we could enrich.
func IncrementProcessEnrichmentHits() {
	processEnrichmentHits.Inc()
}

// SetProcessEnrichmentCacheSize sets the enrichment cache size.
func SetProcessEnrichmentCacheSize(size float64) {
	processEnrichmentLRUCacheSize.Set(size)
}

// IncK8sEventCount increments the number of objects we're receiving from k8s
func IncK8sEventCount(action string, resource string) {
	k8sObjectCounts.With(prometheus.Labels{
		"Action":   action,
		"Resource": resource,
	}).Inc()
}

// SetResourceProcessingDurationForResource sets the duration for how long it takes to process the resource
func SetResourceProcessingDurationForResource(event *central.SensorEvent) {
	metrics.SetResourceProcessingDurationForEvent(k8sObjectProcessingDuration, event, "")
}
