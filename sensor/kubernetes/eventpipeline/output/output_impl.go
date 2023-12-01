package output

import (
	"context"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/sensor/common/detector"
	"github.com/stackrox/rox/sensor/common/message"
	"github.com/stackrox/rox/sensor/common/metrics"
	"github.com/stackrox/rox/sensor/kubernetes/eventpipeline/component"
)

type outputQueueImpl struct {
	innerQueue   chan *component.ResourceEvent
	forwardQueue chan *message.ExpiringMessage
	detector     detector.Detector
	stopSig      concurrency.Signal
}

// Send a ResourceEvent message to the inner queue
func (q *outputQueueImpl) Send(msg *component.ResourceEvent) {
	q.innerQueue <- msg
	metrics.IncOutputChannelSize()
}

// ResponsesC returns the MsgFromSensor channel
func (q *outputQueueImpl) ResponsesC() <-chan *message.ExpiringMessage {
	return q.forwardQueue
}

// Start the outputQueueImpl component
func (q *outputQueueImpl) Start() error {
	go q.runOutputQueue()
	return nil
}

// Stop the outputQueueImpl component
func (q *outputQueueImpl) Stop(_ error) {
	q.stopSig.Signal()
}

func wrapSensorEvent(update *central.SensorEvent) *central.MsgFromSensor {
	return &central.MsgFromSensor{
		Msg: &central.MsgFromSensor_Event{
			Event: update,
		},
	}
}

// runOutputQueue reads messages from the inner queue, forwards them to the forwardQueue channel
// and sends the deployments (if needed) to Detector
func (q *outputQueueImpl) runOutputQueue() {
	for {
		select {
		case <-q.stopSig.Done():
			return
		case msg, more := <-q.innerQueue:
			if !more {
				return
			}

			if msg.Context == nil {
				msg.Context = context.Background()
			}

			for _, resourceUpdates := range msg.ForwardMessages {
				expiringMessage := message.NewExpiring(msg.Context, wrapSensorEvent(resourceUpdates))
				if !expiringMessage.IsExpired() {
					q.forwardQueue <- expiringMessage
				}
			}

			// The order here is important. We rely on the ReprocessDeployment being called before ProcessDeployment to remove the deployments from the deduper.
			q.detector.ReprocessDeployments(msg.ReprocessDeployments...)
			for _, detectorRequest := range msg.DetectorMessages {
				q.detector.ProcessDeployment(msg.Context, detectorRequest.Object, detectorRequest.Action)
			}
			metrics.DecOutputChannelSize()
		}

	}
}
