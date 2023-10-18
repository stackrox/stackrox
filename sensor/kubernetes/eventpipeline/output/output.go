package output

import (
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/sensor/common/detector"
	"github.com/stackrox/rox/sensor/common/message"
	"github.com/stackrox/rox/sensor/kubernetes/eventpipeline/component"
)

// New instantiates a an output Queue component
func New(detector detector.Detector, queueSize int) component.OutputQueue {
	ch := make(chan *component.ResourceEvent, queueSize)
	forwardQueue := make(chan *message.ExpiringMessage, queueSize)
	outputQueue := &outputQueueImpl{
		detector:     detector,
		innerQueue:   ch,
		forwardQueue: forwardQueue,
		stopSig:      concurrency.NewSignal(),
	}
	return outputQueue
}
