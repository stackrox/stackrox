package output

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/sensor/common/detector"
	"github.com/stackrox/rox/sensor/common/message"
	"github.com/stackrox/rox/sensor/common/metrics"
	"github.com/stackrox/rox/sensor/common/pubsub"
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
	if !features.SensorInternalPubSub.Enabled() {
		go q.runOutputQueue()
	}
	return nil
}

// Stop the outputQueueImpl component
func (q *outputQueueImpl) Stop() {
	if features.SensorInternalPubSub.Enabled() {
		// No goroutine was started; signal stopped so the Wait below returns immediately.
		// TODO(ROX-35054): Remove stopper usage once ResponsesC is migrated to PubSub.
		q.stopper.Flow().ReportStopped()
	}
	if !q.stopper.Client().Stopped().IsDone() {
		defer func() {
			_ = q.stopper.Client().Stopped().Wait()
		}()
	}
	q.stopper.Client().Stop()
}

// ProcessResourceEvent is the PubSub callback invoked by the dispatcher when a resolved
// resource event is published by the resolver (ResolvedResourceEventTopic).
func (q *outputQueueImpl) ProcessResourceEvent(event pubsub.Event) error {
	select {
	case <-q.stopper.Flow().StopRequested():
		return nil
	default:
	}
	msg, ok := event.(*component.ResourceEvent)
	if !ok {
		return errors.New("unable to convert event to *component.ResourceEvent")
	}
	q.processMsg(msg)
	return nil
}

func (q *outputQueueImpl) processMsg(msg *component.ResourceEvent) bool {
	if msg.Context == nil {
		msg.Context = context.Background()
	}
	for _, resourceUpdates := range msg.ForwardMessages {
		expiringMessage := message.NewExpiring(msg.Context, wrapSensorEvent(resourceUpdates))
		if !expiringMessage.IsExpired() {
			select {
			case q.forwardQueue <- expiringMessage:
			case <-q.stopper.Flow().StopRequested():
				return false
			}
		}
	}
	// The order here is important. We rely on ReprocessDeployments being called
	// before ProcessDeployment to remove the deployments from the deduper.
	q.detector.ReprocessDeployments(msg.ReprocessDeployments...)
	for _, detectorRequest := range msg.DetectorMessages {
		q.detector.ProcessDeployment(msg.Context, detectorRequest.Object, detectorRequest.Action)
	}
	return true
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
			if !q.processMsg(msg) {
				return
			}
			metrics.DecOutputChannelSize()
		}

	}
}
