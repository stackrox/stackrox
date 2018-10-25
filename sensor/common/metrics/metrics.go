package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/pkg/metrics"
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

	sensorIndicatorChannelFullCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "indicators_channel_indicator_dropped_counter",
		Help:      "A counter of the total number of times we've dropped indicators from the indicators channel because it was full",
	}, []string{"ClusterID"})

	totalNetworkFlowsSentCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "total_network_flows_sent_counter",
		Help:      "A counter of the total number of network flows sent to Central by Sensor",
	}, []string{"ClusterID"})

	totalNetworkFlowsReceivedCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "total_network_flows_sensor_received_counter",
		Help:      "A counter of the total number of network flows received by Sensor from Collector",
	}, []string{"ClusterID"})
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
func RegisterSensorIndicatorChannelFullCounter(clusterID string) {
	sensorIndicatorChannelFullCounter.With(prometheus.Labels{"ClusterID": clusterID}).Inc()
}

// IncrementTotalNetworkFlowsSentCounter registers the total number of flows processed
func IncrementTotalNetworkFlowsSentCounter(clusterID string, numberOfFlows int) {
	totalNetworkFlowsSentCounter.With(prometheus.Labels{"ClusterID": clusterID}).Add(float64(numberOfFlows))
}

// IncrementTotalNetworkFlowsReceivedCounter registers the total number of flows received
func IncrementTotalNetworkFlowsReceivedCounter(clusterID string, numberOfFlows int) {
	totalNetworkFlowsReceivedCounter.With(prometheus.Labels{"ClusterID": clusterID}).Add(float64(numberOfFlows))
}
