package eventpipeline

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/sensor/kubernetes/eventpipeline/component"
)

var (
	log = logging.LoggerForModule()
)

type eventPipeline struct {
	output   component.OutputQueue
	resolver component.PipelineComponent
	listener component.PipelineComponent

	eventsC chan *central.MsgFromSensor
	stopSig concurrency.Signal
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
	// The order is important here, we need to start the components
	// that receive messages from other components first
	if err := p.output.Start(); err != nil {
		return err
	}

	if env.ResyncDisabled.BooleanSetting() {
		if err := p.resolver.Start(); err != nil {
			return err
		}
	}

	if err := p.listener.Start(); err != nil {
		return err
	}

	go p.forwardMessages()
	return nil
}

// Stop implements common.SensorComponent
func (p *eventPipeline) Stop(_ error) {
	defer close(p.eventsC)
	// The order is important here, we need to stop the components
	// that send messages to other components first
	p.listener.Stop(nil)
	if env.ResyncDisabled.BooleanSetting() {
		p.resolver.Stop(nil)
	}
	p.output.Stop(nil)
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
