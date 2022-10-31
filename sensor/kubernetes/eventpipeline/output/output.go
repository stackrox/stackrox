package output

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/sensor/common/detector"
	"github.com/stackrox/rox/sensor/kubernetes/eventpipeline/message"
)

var (
	boundedQueueSize = 100
)

// New Creates a new Queue component
func New(stopSig *concurrency.Signal, detector detector.Detector) message.OutputQueue {
	ch := make(chan *message.ResourceEvent, boundedQueueSize)
	forwardQueue := make(chan *central.MsgFromSensor)
	outputQueue := &outputQueueImpl{
		detector:     detector,
		stopSig:      stopSig,
		innerQueue:   ch,
		forwardQueue: forwardQueue,
	}
	return outputQueue
}
