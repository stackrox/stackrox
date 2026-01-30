package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/sensor/common/pubsub"
)

var (
	// lanePublishOperations tracks all publish operations across lanes.
	// Operations: success, error
	lanePublishOperations = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "pubsub_lane_publish_operations_total",
		Help:      "Total number of pubsub lane publish operations by lane, topic, and operation type",
	}, []string{"lane_id", "topic", "operation"})

	// laneConsumerOperations tracks all publish operations across lanes.
	// Operations: success, error, no_consumers
	laneConsumerOperations = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "pubsub_lane_consumer_operations_total",
		Help:      "Total number of pubsub lane consumer operations by lane, topic, consumer, and operation type",
	}, []string{"lane_id", "topic", "consumer_id", "operation"})

	// laneQueueSize tracks the current number of events in each lane's buffer.
	laneQueueSize = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "pubsub_lane_queue_size_current",
		Help:      "Current number of events waiting in the lane buffer",
	}, []string{"lane_id"})

	// laneEventProcessingDuration tracks the time taken to process events.
	laneEventProcessingDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "pubsub_lane_event_processing_duration_seconds",
		Help:      "Time spent processing an event by each consumer callback",
		Buckets:   []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1, 5},
	}, []string{"lane_id", "topic", "consumer_id", "operation"})

	// consumersCurrent tracks the number of registered consumers per lane/topic.
	consumersCurrent = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "pubsub_consumers_current",
		Help:      "Number of currently registered consumers per lane and topic",
	}, []string{"lane_id", "topic"})
)

func RecordPublishOperation(laneID pubsub.LaneID, topic pubsub.Topic, operation Operation) {
	lanePublishOperations.WithLabelValues(laneID.String(), topic.String(), operation.String()).Inc()
}

func RecordConsumerOperation(laneID pubsub.LaneID, topic pubsub.Topic, consumerID pubsub.ConsumerID, operation Operation) {
	laneConsumerOperations.WithLabelValues(laneID.String(), topic.String(), consumerID.String(), operation.String()).Inc()
}

func SetQueueSize(laneID pubsub.LaneID, size int) {
	laneQueueSize.WithLabelValues(laneID.String()).Set(float64(size))
}

func ObserveProcessingDuration(laneID pubsub.LaneID, topic pubsub.Topic, consumerID pubsub.ConsumerID, duration time.Duration, operation Operation) {
	laneEventProcessingDuration.WithLabelValues(laneID.String(), topic.String(), consumerID.String(), operation.String()).Observe(duration.Seconds())
}

func RecordConsumerCount(laneID pubsub.LaneID, topic pubsub.Topic, count int) {
	consumersCurrent.WithLabelValues(laneID.String(), topic.String()).Set(float64(count))
}

func GetConsumerOperationMetric() *prometheus.CounterVec {
	return laneConsumerOperations
}

func init() {
	prometheus.MustRegister(
		lanePublishOperations,
		laneConsumerOperations,
		laneQueueSize,
		laneEventProcessingDuration,
		consumersCurrent,
	)
}
