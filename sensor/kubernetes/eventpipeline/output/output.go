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

// Queue component that redirects Resource Events and Alerts to the output channel
type Queue interface {
	Send(detectionObject *message.ResourceEvent)
	ResponseC() <-chan *central.MsgFromSensor
}

// New Creates a new Queue component
func New(stopSig *concurrency.Signal, detector detector.Detector) Queue {
	ch := make(chan *message.ResourceEvent, boundedQueueSize)
	forwardQueue := make(chan *central.MsgFromSensor)
	outputQueue := &outputImpl{
		detector:     detector,
		stopSig:      stopSig,
		innerQueue:   ch,
		forwardQueue: forwardQueue,
	}
	go outputQueue.startProcessing()
	return outputQueue
}
