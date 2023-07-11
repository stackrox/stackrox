package output

import (
	"sync/atomic"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/detector"
	"github.com/stackrox/rox/sensor/common/metrics"
	"github.com/stackrox/rox/sensor/kubernetes/eventpipeline/component"
)

var (
	log = logging.LoggerForModule()
)

type outputQueueImpl struct {
	innerQueue   chan *component.ResourceEvent
	forwardQueue chan common.ExpiringSensorMessage
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
func (q *outputQueueImpl) ResponsesC() <-chan common.ExpiringSensorMessage {
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

// runOutputQueue reads messages from the inner queue, forwards them to the forwardQueue channel
// and sends the deployments (if needed) to Detector
func (q *outputQueueImpl) runOutputQueue() {
	for {
		msg, more := <-q.innerQueue
		if !more {
			return
		}

		if msg.Context != nil {
			select {
			case <-msg.Context.Done():
				log.Infof("Message from dispatcher %s dropped (context canceled)", msg.DeploymentTiming.GetDispatcher())
				continue
			default:
			}
		} else {
			log.Warnf("Message has no context: (%+v)", msg)
		}

		for _, resourceUpdates := range msg.ForwardMessages {
			q.forwardQueue <- common.ExpiringSensorMessage{
				Context: msg.Context,
				Message: &central.MsgFromSensor{
					Msg: &central.MsgFromSensor_Event{
						Event: resourceUpdates,
					},
				},
			}
		}

		// The order here is important. We rely on the ReprocessDeployment being called before ProcessDeployment to remove the deployments from the deduper.
		q.detector.ReprocessDeployments(msg.ReprocessDeployments...)
		for _, detectorRequest := range msg.DetectorMessages {
			q.detector.ProcessDeployment(detectorRequest.Object, detectorRequest.Action)
		}
		metrics.DecOutputChannelSize()
	}
}
