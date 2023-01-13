package output

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/sensor/common/detector"
	"github.com/stackrox/rox/sensor/kubernetes/eventpipeline/component"
)

var (
	log = logging.LoggerForModule()
)

type outputQueueImpl struct {
	innerQueue   chan *component.ResourceEvent
	forwardQueue chan *central.MsgFromSensor
	detector     detector.Detector
}

// Send a ResourceEvent message to the inner queue
func (q *outputQueueImpl) Send(msg *component.ResourceEvent) {
	q.innerQueue <- msg
}

// ResponsesC returns the MsgFromSensor channel
func (q *outputQueueImpl) ResponsesC() <-chan *central.MsgFromSensor {
	return q.forwardQueue
}

// Start the outputQueueImpl component
func (q *outputQueueImpl) Start() error {
	go q.runOutputQueue()
	return nil
}

// Stop the outputQueueImpl component
func (q *outputQueueImpl) Stop(_ error) {
	defer close(q.innerQueue)
	defer close(q.forwardQueue)
}

// runOutputQueue reads messages from the inner queue, forwards them to the forwardQueue channel
// and sends the deployments (if needed) to Detector
func (q *outputQueueImpl) runOutputQueue() {
	for {
		msg, more := <-q.innerQueue
		if !more {
			return
		}
		for _, resourceUpdates := range msg.ForwardMessages {
			q.forwardQueue <- &central.MsgFromSensor{
				Msg: &central.MsgFromSensor_Event{
					Event: resourceUpdates,
				},
			}
		}

		// The order here is important. We rely on the ReprocessDeployment being called before ProcessDeployment to remove the deployments from the deduper.
		q.detector.ReprocessDeployments(msg.CompatibilityReprocessDeployments...)
		for _, detectorRequest := range msg.CompatibilityDetectionDeployment {
			q.detector.ProcessDeployment(detectorRequest.Object, detectorRequest.Action)
		}
	}
}
