package output

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/booleanpolicy/augmentedobjs"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/sensor/common/detector"
)

var (
	boundedQueueSize = 100
)

type DetectionObject struct {
	deployment             *storage.Deployment
	images                 []*storage.Image
	networkPoliciesApplied *augmentedobjs.NetworkPoliciesApplied
}

type OutputMessage struct {
	ForwardMessages []*central.SensorEvent

	// DetectionObject should be used by the new path
	DetectionObject *DetectionObject

	Action central.ResourceAction
	// CompatibilityDetectionDeployment should be used by old handlers
	// and its here for retrocompatibility reasons.
	// This property should be removed in the future and only the
	// DetectionObject should be sent
	CompatibilityDetectionDeployment *storage.Deployment
}

type Queue interface {
	Send(detectionObject *OutputMessage)
	ResponseC() <-chan *central.MsgFromSensor
}

func New(stopSig *concurrency.Signal, detector detector.Detector) Queue {
	ch := make(chan *OutputMessage, boundedQueueSize)
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
