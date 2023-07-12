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
	"github.com/stackrox/rox/sensor/common/reprocessor"
	"github.com/stackrox/rox/sensor/common/store/resolver"
	"github.com/stackrox/rox/sensor/kubernetes/eventpipeline/component"
	"github.com/stackrox/rox/sensor/kubernetes/listener/resources"
)

var (
	log = logging.LoggerForModule()
)

type eventPipeline struct {
	output        component.OutputQueue
	resolver      component.Resolver
	listener      component.Listener
	detector      detector.Detector
	reprocessor   reprocessor.Handler
	storeProvider *resources.InMemoryStoreProvider

	offlineMode *atomic.Bool

	eventsC chan *central.MsgFromSensor
	stopSig concurrency.Signal

	pipelineContext context.Context
	cancelFunction  context.CancelFunc
	contextMutex    *sync.RWMutex
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

	go p.forwardMessages()
	return nil
}

func (p *eventPipeline) invalidateCurrentContext() {
	p.contextMutex.Lock()
	defer p.contextMutex.Unlock()
	p.cancelFunction()
}

func (p *eventPipeline) swapContext() {
	p.contextMutex.Lock()
	defer p.contextMutex.Unlock()

	p.pipelineContext, p.cancelFunction = context.WithCancel(context.Background())
}

func (p *eventPipeline) getContext() context.Context {
	p.contextMutex.RLock()
	p.contextMutex.RUnlock()

	return p.pipelineContext
}

// Stop implements common.SensorComponent
func (p *eventPipeline) Stop(_ error) {
	defer close(p.eventsC)
	// The order is important here, we need to stop the components
	// that send messages to other components first
	p.invalidateCurrentContext()
	p.listener.Stop(nil)
	if env.ResyncDisabled.BooleanSetting() {
		p.resolver.Stop(nil)
	}
	p.output.Stop(nil)
	p.stopSig.Signal()
}

func (p *eventPipeline) Notify(event common.SensorComponentEvent) {
	log.Infof("Received notify: %s", event)
	switch event {
	case common.SensorComponentEventCentralReachable:
		// Start listening to events if not yet listening
		if p.offlineMode.CompareAndSwap(true, false) {
			log.Infof("Connection established: Starting Kubernetes listener")
			p.swapContext()
			p.listener.SetContext(p.getContext())
			if err := p.listener.Start(); err != nil {
				log.Fatalf("Failed to start listener component. Sensor cannot run without listening to Kubernetes events: %s", err)
			}
		}
	case common.SensorComponentEventOfflineMode:
		// Stop listening to events
		if p.offlineMode.CompareAndSwap(false, true) {
			p.invalidateCurrentContext()
			p.listener.Stop(errors.New("gRPC connection stopped"))
			p.storeProvider.CleanupStores()
		}
	}
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
			if msg.Message.GetEvent().GetDeployment().GetNamespace() == "sensor-integration" {
				log.Infof("PIPELINE OUTPUT(%s): %s", msg.Message.GetEvent().GetTiming().GetDispatcher(), msg.Message.GetEvent().GetDeployment().GetName())
			}

			select {
			case p.eventsC <- msg.Message:
			case <-msg.Context.Done():
			}
		}
	}
}

func (p *eventPipeline) processPolicySync(sync *central.PolicySync) error {
	log.Debug("PolicySync message received from central")
	return p.detector.ProcessPolicySync(sync)
}

func (p *eventPipeline) processReassessPolicies() error {
	log.Debug("ReassessPolicies message received from central")
	if err := p.detector.ProcessReassessPolicies(); err != nil {
		return err
	}
	if env.ResyncDisabled.BooleanSetting() {
		message := component.NewEvent()
		// TODO(ROX-14310): Add WithSkipResolving to the DeploymentReference (Revert: https://github.com/stackrox/stackrox/pull/5551)
		message.AddDeploymentReference(resolver.ResolveAllDeployments(),
			component.WithForceDetection())
		message.Context = p.getContext()
		p.resolver.Send(message)
	}
	return nil
}

func (p *eventPipeline) processReprocessDeployments() error {
	log.Debug("ReprocessDeployments message received from central")
	if err := p.detector.ProcessReprocessDeployments(); err != nil {
		return err
	}
	if env.ResyncDisabled.BooleanSetting() {
		message := component.NewEvent()
		// TODO(ROX-14310): Add WithSkipResolving to the DeploymentReference (Revert: https://github.com/stackrox/stackrox/pull/5551)
		message.AddDeploymentReference(resolver.ResolveAllDeployments(),
			component.WithForceDetection())
		message.Context = p.getContext()
		p.resolver.Send(message)
	}
	return nil
}

func (p *eventPipeline) processUpdatedImage(image *storage.Image) error {
	log.Debugf("UpdatedImage message received from central: image name: %s, number of components: %d", image.GetName().GetFullName(), image.GetComponents())
	if err := p.detector.ProcessUpdatedImage(image); err != nil {
		return err
	}
	if env.ResyncDisabled.BooleanSetting() {
		message := component.NewEvent()
		message.AddDeploymentReference(resolver.ResolveDeploymentsByImages(image),
			component.WithForceDetection(),
			component.WithSkipResolving())
		message.Context = p.getContext()
		p.resolver.Send(message)
	}
	return nil
}

func (p *eventPipeline) processReprocessDeployment(req *central.ReprocessDeployment) error {
	log.Debug("ReprocessDeployment message received from central")
	if err := p.reprocessor.ProcessReprocessDeployments(req); err != nil {
		return err
	}
	if env.ResyncDisabled.BooleanSetting() {
		message := component.NewEvent()
		message.AddDeploymentReference(resolver.ResolveDeploymentIds(req.GetDeploymentIds()...),
			component.WithForceDetection(),
			component.WithSkipResolving())
		message.Context = p.getContext()
		p.resolver.Send(message)
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
		message := component.NewEvent()
		message.AddDeploymentReference(resolver.ResolveDeploymentsByImages(keys...),
			component.WithForceDetection(),
			component.WithSkipResolving())
		message.Context = p.getContext()
		p.resolver.Send(message)
	}
	return nil
}
