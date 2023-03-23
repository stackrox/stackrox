package eventpipeline

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/detector"
	"github.com/stackrox/rox/sensor/common/store/resolver"
	"github.com/stackrox/rox/sensor/kubernetes/eventpipeline/component"
)

var (
	log = logging.LoggerForModule()
)

type eventPipeline struct {
	output   component.OutputQueue
	resolver component.Resolver
	listener component.PipelineComponent
	detector detector.Detector

	eventsC chan *central.MsgFromSensor
	stopSig concurrency.Signal
}

// Capabilities implements common.SensorComponent
func (*eventPipeline) Capabilities() []centralsensor.SensorCapability {
	return nil
}

// ProcessMessage implements common.SensorComponent
func (p *eventPipeline) ProcessMessage(msg *central.MsgToSensor) error {
	switch {
	case msg.GetPolicySync() != nil:
		return p.processPolicySync(msg.GetPolicySync())
	case msg.GetReassessPolicies() != nil:
		return p.processReassessPolicies()
	case msg.GetUpdatedImage() != nil:
		return p.processUpdatedImage(msg.GetUpdatedImage())
	case msg.GetReprocessDeployments() != nil:
		return p.processReprocessDeployments()
	}
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

func (p *eventPipeline) Notify(common.SensorComponentEvent) {}

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

func (p *eventPipeline) processPolicySync(sync *central.PolicySync) error {
	return p.detector.ProcessPolicySync(sync)
}

func (p *eventPipeline) processReassessPolicies() error {
	if err := p.detector.ProcessReassessPolicies(); err != nil {
		return err
	}
	if env.ResyncDisabled.BooleanSetting() {
		message := component.NewEvent()
		message.AddDeploymentReference(resolver.ResolveAllDeployments(), central.ResourceAction_UPDATE_RESOURCE, true, true)
		log.Debugf("Reassess message to the Resolver: %+v", message)
		p.resolver.Send(message)
	}
	return nil
}

func (p *eventPipeline) processReprocessDeployments() error {
	if err := p.detector.ProcessReprocessDeployments(); err != nil {
		return err
	}
	if env.ResyncDisabled.BooleanSetting() {
		message := component.NewEvent()
		message.AddDeploymentReference(resolver.ResolveAllDeployments(), central.ResourceAction_UPDATE_RESOURCE, true, true)
		log.Debugf("Reprocess message to the Resolver: %+v", message)
		p.resolver.Send(message)
	}
	return nil
}

func (p *eventPipeline) processUpdatedImage(image *storage.Image) error {
	if err := p.detector.ProcessUpdatedImage(image); err != nil {
		return err
	}
	if env.ResyncDisabled.BooleanSetting() {
		message := component.NewEvent()
		message.AddDeploymentReference(resolver.ResolveDeploymentsByImage(image), central.ResourceAction_UPDATE_RESOURCE, true, true)
		log.Debugf("Updated Image message to the Resolver: %+v", message)
		p.resolver.Send(message)
	}
	return nil
}
