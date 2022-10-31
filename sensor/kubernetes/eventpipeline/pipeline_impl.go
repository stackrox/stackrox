package eventpipeline

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/sensor/kubernetes/eventpipeline/message"
)

var (
	log = logging.LoggerForModule()
)

type eventPipeline struct {
	output     message.OutputQueue
	components []message.PipelineComponent

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
	for _, c := range p.components {
		if err := c.Start(); err != nil {
			return err
		}
	}
	go p.forwardMessages()
	return nil
}

// Stop implements common.SensorComponent
func (p *eventPipeline) Stop(err error) {
	defer close(p.eventsC)
	for _, c := range p.components {
		c.Stop(err)
	}
	p.stopSig.Signal()
}

// forwardMessages from listener component to responses channel
func (p *eventPipeline) forwardMessages() {
	for {
		select {
		case <-p.stopSig.Done():
			return
		case msg, more := <-p.output.ResponsesC():
			if !more {
				log.Error("Output component channel closed")
				return
			}
			p.eventsC <- msg
		}
	}
}
