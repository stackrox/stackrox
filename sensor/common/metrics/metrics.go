package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/pkg/metrics"
)

var (
	// Panics encountered
	panicCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.Namespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "panic_counter",
		Help:      "Number of panic calls within Sensor.",
	}, []string{"FunctionName"})

	signalToIndicatorCreateLagGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: metrics.Namespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "signal_to_indicator_lag",
		Help:      "Nanoseconds between the signal emission timestamp and the creation time of an indicator message",
	}, []string{"ClusterID"})

	signalToIndicatorEmitLagGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: metrics.Namespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "signal_to_indicator_send_lag",
		Help:      "Nanoseconds between the signal emission timestamp and the emission time of an indicator message",
	}, []string{"ClusterID"})

	sensorIndicatorChannelFullCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.Namespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "indicators_channel_indicator_dropped_counter",
		Help:      "A counter of the total number of times we've dropped indicators from the indicators channel because it was full",
	}, []string{"ClusterID"})
)

// IncrementPanicCounter increments the number of panic calls seen in a function
func IncrementPanicCounter(functionName string) {
	panicCounter.With(prometheus.Labels{"FunctionName": functionName}).Inc()
}

// RegisterSignalToIndicatorCreateLag registers the lag between a collector signal timestamp and the create timestamp of an indicator
func RegisterSignalToIndicatorCreateLag(clusterID string, lag float64) {
	signalToIndicatorCreateLagGauge.With(prometheus.Labels{"ClusterID": clusterID}).Set(lag)
}

// RegisterSignalToIndicatorEmitLag registers the lag between a collector signal timestamp and the emit timestamp of an indicator
func RegisterSignalToIndicatorEmitLag(clusterID string, lag float64) {
	signalToIndicatorEmitLagGauge.With(prometheus.Labels{"ClusterID": clusterID}).Set(lag)
}

// RegisterSensorIndicatorChannelFullCounter increments indicator channel drops when the channel is full
func RegisterSensorIndicatorChannelFullCounter(clusterID string) {
	sensorIndicatorChannelFullCounter.With(prometheus.Labels{"ClusterID": clusterID}).Inc()
}
