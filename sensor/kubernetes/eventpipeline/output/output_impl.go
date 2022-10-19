package output

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/sensor/common/detector"
)

type outputImpl struct {
	innerQueue   chan *OutputMessage
	stopSig      *concurrency.Signal
	forwardQueue chan *central.MsgFromSensor
	detector     detector.Detector
}

func (q *outputImpl) Send(msg *OutputMessage) {
	q.innerQueue <- msg
}

func (q *outputImpl) ResponseC() <-chan *central.MsgFromSensor {
	return q.forwardQueue
}

func (q *outputImpl) startProcessing() {
	for {
		select {
		case <-q.stopSig.Done():
			return
		case msg, more := <-q.innerQueue:
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

			if msg.CompatibilityDetectionDeployment != nil {
				q.detector.ProcessDeployment(msg.CompatibilityDetectionDeployment, msg.Action)
			}
		}
	}
}
