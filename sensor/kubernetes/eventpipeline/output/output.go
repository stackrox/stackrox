package output

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/sensor/common/detector"
	"github.com/stackrox/rox/sensor/kubernetes/eventpipeline/component"
)

var (
	boundedQueueSize = 100
)

// New Creates a new Queue component
func New(stopSig *concurrency.Signal, detector detector.Detector) component.OutputQueue {
	ch := make(chan *component.ResourceEvent, boundedQueueSize)
	forwardQueue := make(chan *central.MsgFromSensor)
	outputQueue := &outputQueueImpl{
		detector:     detector,
		stopSig:      stopSig,
		innerQueue:   ch,
		forwardQueue: forwardQueue,
	}
	return outputQueue
}
