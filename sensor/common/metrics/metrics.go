package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/branding"
	"github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/version"
	"github.com/stackrox/rox/sensor/common/centralid"
	"github.com/stackrox/rox/sensor/common/clusterid"
	"github.com/stackrox/rox/sensor/common/installmethod"
)

var (
	// Panics encountered
	panicCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "panic_counter",
		Help:      "Number of panic calls within Sensor.",
	}, []string{"FunctionName"})

	detectorDedupeCacheHits = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "detector_dedupe_cache_hits",
		Help:      "A counter of the total number of times we've deduped deployments in the detector",
	})

	detectorDeploymentProcessed = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "detector_deployment_processed",
		Help:      "A counter of the total number of times we've processed deployments in the detector",
	})

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

	totalProcessesSentCounter = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "total_processes_sent_counter",
		Help:      "A counter of the total number of processes sent to Central by Sensor",
	})

	totalProcessesReceivedCounter = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "total_processes_received_counter",
		Help:      "A counter of the total number of processes received by Sensor from Collector",
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

	resolverChannelSize = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "resolver_channel_size",
		Help:      "A gauge to track the resolver channel size",
	})

	outputChannelSize = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "output_channel_size",
		Help:      "A gauge to track the output channel size",
	})

	telemetryLabels = prometheus.Labels{
		"branding":       branding.GetProductNameShort(),
		"build":          metrics.GetBuildType(),
		"sensor_version": version.GetMainVersion(),
	}

	telemetryInfo = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace:   metrics.PrometheusNamespace,
			Subsystem:   metrics.SensorSubsystem.String(),
			Name:        "info",
			Help:        "Telemetry information about Sensor",
			ConstLabels: telemetryLabels,
		},
		[]string{"central_id", "hosting", "install_method", "sensor_id"},
	)

	telemetrySecuredNodes = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace:   metrics.PrometheusNamespace,
			Subsystem:   metrics.SensorSubsystem.String(),
			Name:        "secured_nodes",
			Help:        "The number of nodes secured by Sensor",
			ConstLabels: telemetryLabels,
		},
		[]string{"central_id", "hosting", "install_method", "sensor_id"},
	)

	telemetrySecuredVCPU = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace:   metrics.PrometheusNamespace,
			Subsystem:   metrics.SensorSubsystem.String(),
			Name:        "secured_vcpus",
			Help:        "The number of vCPUs secured by Sensor",
			ConstLabels: telemetryLabels,
		},
		[]string{"central_id", "hosting", "install_method", "sensor_id"},
	)
)

// IncrementDetectorCacheHit increments the number of deployments deduped by the detector
func IncrementDetectorCacheHit() {
	detectorDedupeCacheHits.Inc()
}

// IncrementDetectorDeploymentProcessed increments the number of deployments processed by the detector
func IncrementDetectorDeploymentProcessed() {
	detectorDeploymentProcessed.Inc()
}

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

// IncrementTotalProcessesSentCounter increments the total number of endpoints sent
func IncrementTotalProcessesSentCounter(numberOfProcesses int) {
	totalProcessesSentCounter.Add(float64(numberOfProcesses))
}

// IncrementTotalProcessesReceivedCounter increments the total number of endpoints received
func IncrementTotalProcessesReceivedCounter(numberOfProcesses int) {
	totalProcessesReceivedCounter.Add(float64(numberOfProcesses))
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

// IncResolverChannelSize increases the resolverChannel by 1
func IncResolverChannelSize() {
	resolverChannelSize.Inc()
}

// DecResolverChannelSize decreases the resolverChannel by 1
func DecResolverChannelSize() {
	resolverChannelSize.Dec()
}

// IncOutputChannelSize increases the outputChannel by 1
func IncOutputChannelSize() {
	outputChannelSize.Inc()
}

// DecOutputChannelSize decreases the outputChannel by 1
func DecOutputChannelSize() {
	outputChannelSize.Dec()
}

// SetTelemetryMetrics sets the cluster metrics for the telemetry metrics.
func SetTelemetryMetrics(cm *central.ClusterMetrics) {
	labels := []string{
		centralid.Get(),
		getHosting(),
		installmethod.Get(),
		clusterid.GetNoWait(),
	}

	telemetryInfo.Reset()
	telemetryInfo.WithLabelValues(labels...).Set(1)

	telemetrySecuredNodes.Reset()
	telemetrySecuredNodes.WithLabelValues(labels...).Set(float64(cm.GetNodeCount()))

	telemetrySecuredVCPU.Reset()
	telemetrySecuredVCPU.WithLabelValues(labels...).Set(float64(cm.GetCpuCapacity()))
}
