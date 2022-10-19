package eventpipeline

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/kubernetes/eventpipeline/output"
)

var (
	log = logging.LoggerForModule()
)

type eventPipeline struct {
	listener common.SensorComponent
	output   output.Queue

	eventsC chan *central.MsgFromSensor
	stopSig *concurrency.Signal
}

// Capabilities implements common.SensorComponent
func (*eventPipeline) Capabilities() []centralsensor.SensorCapability {
	return nil
}

// ProcessMessage implements common.SensorComponent
func (*eventPipeline) ProcessMessage(msg *central.MsgToSensor) error {
	return nil
}

// ResponsesC implements common.SensorComponent
func (p *eventPipeline) ResponsesC() <-chan *central.MsgFromSensor {
	return p.eventsC
}

// Start implements common.SensorComponent
func (p *eventPipeline) Start() error {
	log.Info("STARTING EVENT PIPELINE")
	if err := p.listener.Start(); err != nil {
		return err
	}
	go p.forwardMessages()
	return nil
}

// Stop implements common.SensorComponent
func (p *eventPipeline) Stop(err error) {
	p.listener.Stop(err)
}

// forwardMessages from listener component to responses channel
// TODO: Remove this and refactor listeners so they send message to the pipeline queue instead.
func (p *eventPipeline) forwardMessages() {
	log.Info("starting message forwarding")
	defer func() { 
		log.Info("stopping message forward")
	}()

	for {
		select {
		case <-p.stopSig.Done():
			return
		case msg, more := <-p.output.ResponseC():
			if !more {
				// TODO: Add warning / error / log
				return
			}
			p.eventsC <- msg
		case msg, more := <-p.listener.ResponsesC():
			log.Info("forwarding message from listener")
			if !more {
				// TODO: Add warning / error / log
				return
			}
			p.eventsC <- msg
		}

	}

}
