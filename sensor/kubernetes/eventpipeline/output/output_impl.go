package output

import (
	"context"
	"sync/atomic"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/sensor/common/detector"
	"github.com/stackrox/rox/sensor/common/message"
	"github.com/stackrox/rox/sensor/common/metrics"
	"github.com/stackrox/rox/sensor/kubernetes/eventpipeline/component"
)

var (
	log = logging.LoggerForModule()
)

type outputQueueImpl struct {
	innerQueue   chan *component.ResourceEvent
	forwardQueue chan *message.ExpiringMessage
	detector     detector.Detector
	stopped      *atomic.Bool
}

// Send a ResourceEvent message to the inner queue
func (q *outputQueueImpl) Send(msg *component.ResourceEvent) {
	if !q.stopped.Load() {
		q.innerQueue <- msg
		metrics.IncOutputChannelSize()
	}
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
	defer close(q.innerQueue)
	defer close(q.forwardQueue)
	q.stopped.Store(true)
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
		msg, more := <-q.innerQueue
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

		// TODO(ROX-17326): Don't process message in the detector if message expired
		// The order here is important. We rely on the ReprocessDeployment being called before ProcessDeployment to remove the deployments from the deduper.
		q.detector.ReprocessDeployments(msg.ReprocessDeployments...)
		for _, detectorRequest := range msg.DetectorMessages {
			q.detector.ProcessDeployment(msg.Context, detectorRequest.Object, detectorRequest.Action)
		}
		metrics.DecOutputChannelSize()
	}
}
