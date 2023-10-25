package eventpipeline

import (
	"context"
	"sync/atomic"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/detector"
	"github.com/stackrox/rox/sensor/common/message"
	"github.com/stackrox/rox/sensor/common/reprocessor"
	"github.com/stackrox/rox/sensor/common/store/resolver"
	"github.com/stackrox/rox/sensor/kubernetes/eventpipeline/component"
)

var (
	log = logging.LoggerForModule()
)

type eventPipeline struct {
	output      component.OutputQueue
	resolver    component.Resolver
	listener    component.ContextListener
	detector    detector.Detector
	reprocessor reprocessor.Handler

	offlineMode *atomic.Bool

	eventsC chan *message.ExpiringMessage
	stopSig concurrency.Signal

	contextMtx    sync.Mutex
	context       context.Context
	cancelContext context.CancelFunc
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
	case msg.GetReprocessDeployment() != nil:
		return p.processReprocessDeployment(msg.GetReprocessDeployment())
	case msg.GetInvalidateImageCache() != nil:
		return p.processInvalidateImageCache(msg.GetInvalidateImageCache())
	}
	return nil
}

// ResponsesC implements common.SensorComponent
func (p *eventPipeline) ResponsesC() <-chan *message.ExpiringMessage {
	return p.eventsC
}

func (p *eventPipeline) stopCurrentContext() {
	p.contextMtx.Lock()
	defer p.contextMtx.Unlock()
	if p.cancelContext != nil {
		p.cancelContext()
	}
}

func (p *eventPipeline) getCurrentContext() context.Context {
	p.contextMtx.Lock()
	defer p.contextMtx.Unlock()
	return p.context
}

func (p *eventPipeline) createNewContext() {
	p.contextMtx.Lock()
	defer p.contextMtx.Unlock()
	p.context, p.cancelContext = context.WithCancel(context.Background())
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

	go p.forwardMessages()
	return nil
}

// Stop implements common.SensorComponent
func (p *eventPipeline) Stop(_ error) {
	// The order is important here, we need to stop the components
	// that send messages to other components first
	p.listener.Stop(nil)
	if env.ResyncDisabled.BooleanSetting() {
		p.resolver.Stop(nil)
	}
	p.output.Stop(nil)
	p.stopSig.Signal()
}

func (p *eventPipeline) Notify(e common.SensorComponentEvent) {
	log.Info(common.LogSensorComponentEvent(e))
	switch e {
	case common.SensorComponentEventCentralReachable:
		// Start listening to events if not yet listening
		if p.offlineMode.CompareAndSwap(true, false) {
			log.Info("Connection established: Starting Kubernetes listener")
			// TODO(ROX-18613): use contextProvider to provide context for listener
			p.createNewContext()
			if err := p.listener.StartWithContext(p.context); err != nil {
				log.Fatalf("Failed to start listener component. Sensor cannot run without listening to Kubernetes events: %s", err)
			}
		}
	case common.SensorComponentEventOfflineMode:
		// Stop listening to events
		if p.offlineMode.CompareAndSwap(false, true) {
			p.stopCurrentContext()
			p.listener.Stop(errors.New("gRPC connection stopped"))
		}
	}
}

// forwardMessages from listener component to responses channel
func (p *eventPipeline) forwardMessages() {
	defer close(p.eventsC)
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
	log.Debug("PolicySync message received from central")
	return p.detector.ProcessPolicySync(p.getCurrentContext(), sync)
}

func (p *eventPipeline) processReassessPolicies() error {
	log.Debug("ReassessPolicies message received from central")
	if err := p.detector.ProcessReassessPolicies(); err != nil {
		return err
	}
	if env.ResyncDisabled.BooleanSetting() {
		msg := component.NewEvent()
		// TODO(ROX-14310): Add WithSkipResolving to the DeploymentReference (Revert: https://github.com/stackrox/stackrox/pull/5551)
		msg.AddDeploymentReference(resolver.ResolveAllDeployments(),
			component.WithForceDetection())
		msg.Context = p.getCurrentContext()
		p.resolver.Send(msg)
	}
	return nil
}

func (p *eventPipeline) processReprocessDeployments() error {
	log.Debug("ReprocessDeployments message received from central")
	if err := p.detector.ProcessReprocessDeployments(); err != nil {
		return err
	}
	if env.ResyncDisabled.BooleanSetting() {
		msg := component.NewEvent()
		// TODO(ROX-14310): Add WithSkipResolving to the DeploymentReference (Revert: https://github.com/stackrox/stackrox/pull/5551)
		msg.AddDeploymentReference(resolver.ResolveAllDeployments(),
			component.WithForceDetection())
		msg.Context = p.getCurrentContext()
		p.resolver.Send(msg)
	}
	return nil
}

func (p *eventPipeline) processUpdatedImage(image *storage.Image) error {
	log.Debugf("UpdatedImage message received from central: image name: %s, number of components: %d", image.GetName().GetFullName(), image.GetComponents())
	if err := p.detector.ProcessUpdatedImage(image); err != nil {
		return err
	}
	if env.ResyncDisabled.BooleanSetting() {
		msg := component.NewEvent()
		msg.AddDeploymentReference(resolver.ResolveDeploymentsByImages(image),
			component.WithForceDetection(),
			component.WithSkipResolving())
		msg.Context = p.getCurrentContext()
		p.resolver.Send(msg)
	}
	return nil
}

func (p *eventPipeline) processReprocessDeployment(req *central.ReprocessDeployment) error {
	log.Debug("ReprocessDeployment message received from central")
	if err := p.reprocessor.ProcessReprocessDeployments(req); err != nil {
		return err
	}
	if env.ResyncDisabled.BooleanSetting() {
		msg := component.NewEvent()
		msg.AddDeploymentReference(resolver.ResolveDeploymentIds(req.GetDeploymentIds()...),
			component.WithForceDetection(),
			component.WithSkipResolving())
		msg.Context = p.getCurrentContext()
		p.resolver.Send(msg)
	}
	return nil
}

func (p *eventPipeline) processInvalidateImageCache(req *central.InvalidateImageCache) error {
	log.Debug("InvalidateImageCache message received from central")
	if err := p.reprocessor.ProcessInvalidateImageCache(req); err != nil {
		return err
	}
	if env.ResyncDisabled.BooleanSetting() {
		keys := make([]*storage.Image, len(req.GetImageKeys()))
		for i, image := range req.GetImageKeys() {
			keys[i] = &storage.Image{
				Id: image.GetImageId(),
				Name: &storage.ImageName{
					FullName: image.GetImageFullName(),
				},
			}
		}
		msg := component.NewEvent()
		msg.AddDeploymentReference(resolver.ResolveDeploymentsByImages(keys...),
			component.WithForceDetection(),
			component.WithSkipResolving())
		msg.Context = p.getCurrentContext()
		p.resolver.Send(msg)
	}
	return nil
}
