package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/pkg/metrics"
)

var (
	// laneEventOperations tracks all event operations across lanes.
	// Operations: published, processed, publish_error, consumer_error, no_consumers
	laneEventOperations = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "pubsub_lane_event_operations_total",
		Help:      "Total number of pubsub lane event operations by lane, topic, and operation type",
	}, []string{"lane_id", "topic", "operation"})

	// laneQueueSize tracks the current number of events in each lane's buffer.
	laneQueueSize = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "pubsub_lane_queue_size",
		Help:      "Current number of events waiting in the lane buffer",
	}, []string{"lane_id"})

	// laneEventProcessingDuration tracks the time taken to process events.
	laneEventProcessingDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "pubsub_lane_event_processing_duration_seconds",
		Help:      "Time spent processing an event through all consumer callbacks",
		Buckets:   []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1, 5},
	}, []string{"lane_id", "topic"})

	// consumersCount tracks the number of registered consumers per lane/topic.
	consumersCount = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "pubsub_consumers_count",
		Help:      "Number of currently registered consumers per lane and topic",
	}, []string{"lane_id", "topic"})
)
