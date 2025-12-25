package metrics

import (
	"time"

	"github.com/stackrox/rox/sensor/common/pubsub"
)

// Recorder handles metrics recording for pubsub operations.
type Recorder interface {
	// RecordOperation records a lane operation
	RecordOperation(laneID pubsub.LaneID, topic pubsub.Topic, operation Operation)
	// SetQueueSize sets the current queue size for a lane
	SetQueueSize(laneID pubsub.LaneID, size int)
	// ObserveProcessingDuration records the time taken to process an event
	ObserveProcessingDuration(laneID pubsub.LaneID, topic pubsub.Topic, duration time.Duration)
	// RecordConsumerCount sets the number of registered consumers for a lane/topic
	RecordConsumerCount(laneID pubsub.LaneID, topic pubsub.Topic, count int)
}

// DefaultRecorder is the production Prometheus-backed recorder.
var DefaultRecorder Recorder = &prometheusRecorder{}

// prometheusRecorder implements Recorder using Prometheus metrics.
type prometheusRecorder struct{}

func (p *prometheusRecorder) RecordOperation(laneID pubsub.LaneID, topic pubsub.Topic, operation Operation) {
	laneEventOperations.WithLabelValues(laneID.String(), topic.String(), operation.String()).Inc()
}

func (p *prometheusRecorder) SetQueueSize(laneID pubsub.LaneID, size int) {
	laneQueueSize.WithLabelValues(laneID.String()).Set(float64(size))
}

func (p *prometheusRecorder) ObserveProcessingDuration(laneID pubsub.LaneID, topic pubsub.Topic, duration time.Duration) {
	laneEventProcessingDuration.WithLabelValues(laneID.String(), topic.String()).Observe(duration.Seconds())
}

func (p *prometheusRecorder) RecordConsumerCount(laneID pubsub.LaneID, topic pubsub.Topic, count int) {
	consumersCount.WithLabelValues(laneID.String(), topic.String()).Set(float64(count))
}

// NoOpRecorder is a no-op implementation for testing.
type NoOpRecorder struct{}

func (n *NoOpRecorder) RecordOperation(pubsub.LaneID, pubsub.Topic, Operation) {}

func (n *NoOpRecorder) SetQueueSize(pubsub.LaneID, int) {}

func (n *NoOpRecorder) ObserveProcessingDuration(pubsub.LaneID, pubsub.Topic, time.Duration) {}

func (n *NoOpRecorder) RecordConsumerCount(pubsub.LaneID, pubsub.Topic, int) {}
