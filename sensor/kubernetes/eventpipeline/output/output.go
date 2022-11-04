package output

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/sensor/common/detector"
	"github.com/stackrox/rox/sensor/kubernetes/eventpipeline/component"
)

// New instantiates a an output Queue component
func New(detector detector.Detector, queueSize int) component.OutputQueue {
	ch := make(chan *component.ResourceEvent, queueSize)
	forwardQueue := make(chan *central.MsgFromSensor)
	outputQueue := &outputQueueImpl{
		detector:     detector,
		innerQueue:   ch,
		forwardQueue: forwardQueue,
	}
	return outputQueue
}
