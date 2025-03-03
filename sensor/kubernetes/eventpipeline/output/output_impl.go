package output

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/sensor/common/detector"
	"github.com/stackrox/rox/sensor/common/message"
	"github.com/stackrox/rox/sensor/common/metrics"
	"github.com/stackrox/rox/sensor/common/trace"
	"github.com/stackrox/rox/sensor/kubernetes/eventpipeline/component"
)

type outputQueueImpl struct {
	innerQueue   chan *component.ResourceEvent
	forwardQueue chan *message.ExpiringMessage
	detector     detector.Detector
	stopper      concurrency.Stopper
}

// Send a ResourceEvent to outside the pipeline. This will trigger alert detection if component.ResourceEvent
// has any DetectorMessages (component.DeploytimeDetectionRequest).
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
	if !q.stopper.Client().Stopped().IsDone() {
		defer func() {
			_ = q.stopper.Client().Stopped().Wait()
		}()
	}
	q.stopper.Client().Stop()
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
	defer q.stopper.Flow().ReportStopped()
	for {
		select {
		case <-q.stopper.Flow().StopRequested():
			return
		case msg, more := <-q.innerQueue:
			if !more {
				return
			}

			if msg.Context == nil {
				msg.Context = trace.Context()
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
