package eventpipeline

import (
	"context"
	"sync/atomic"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/detector"
	"github.com/stackrox/rox/sensor/common/message"
	"github.com/stackrox/rox/sensor/common/reprocessor"
	"github.com/stackrox/rox/sensor/common/store/resolver"
	"github.com/stackrox/rox/sensor/common/trace"
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
	stopper concurrency.Stopper

	contextMtx    sync.Mutex
	context       context.Context
	cancelContext context.CancelFunc
}

func (p *eventPipeline) Name() string {
	return "eventpipeline.eventPipeline"
}

// Capabilities implements common.SensorComponent
func (*eventPipeline) Capabilities() []centralsensor.SensorCapability {
	return nil
}

// ProcessMessage implements common.SensorComponent
func (p *eventPipeline) ProcessMessage(_ context.Context, msg *central.MsgToSensor) error {
	switch {
	case msg.GetPolicySync() != nil:
		return p.processPolicySync(msg.GetPolicySync())
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
	p.context, p.cancelContext = context.WithCancel(trace.Background())
}

// Start implements common.SensorComponent
func (p *eventPipeline) Start() error {
	// The order is important here, we need to start the components
	// that receive messages from other components first
	if err := p.output.Start(); err != nil {
		return errors.Wrap(err, "starting output component")
	}

	if err := p.resolver.Start(); err != nil {
		return errors.Wrap(err, "starting resolver component")
	}

	go p.forwardMessages()
	return nil
}

// Stop implements common.SensorComponent
func (p *eventPipeline) Stop() {
	if !p.stopper.Client().Stopped().IsDone() {
		defer func() {
			_ = p.stopper.Client().Stopped().Wait()
		}()
	}
	// The order is important here, we need to stop the components
	// that send messages to other components first
	p.listener.Stop()
	p.resolver.Stop()
	p.output.Stop()
	p.stopper.Client().Stop()
}

func (p *eventPipeline) Notify(e common.SensorComponentEvent) {
	log.Info(common.LogSensorComponentEvent(e, "Event Pipeline"))
	switch e {
	case common.SensorComponentEventCentralReachable:
		// Start listening to events if not yet listening
		if p.offlineMode.CompareAndSwap(true, false) {
			log.Info("Connection established: Starting Kubernetes listener")
			// Stopping the listener here will allow Sensor to maintain the stores populated while offline.
			// This is needed to capture runtime events in offline mode.
			p.listener.Stop()
			// TODO(ROX-18613): use contextProvider to provide context for listener
			p.createNewContext()
			if err := p.listener.StartWithContext(p.context); err != nil {
				log.Fatalf("Failed to start listener component. Sensor cannot run without listening to Kubernetes events: %s", err)
			}
		}
	case common.SensorComponentEventOfflineMode:
		// Cancel the current context of the listeners.
		if p.offlineMode.CompareAndSwap(false, true) {
			p.stopCurrentContext()
		}
	}
}

// forwardMessages from listener component to responses channel
func (p *eventPipeline) forwardMessages() {
	defer close(p.eventsC)
	defer p.stopper.Flow().ReportStopped()
	for {
		select {
		case <-p.stopper.Flow().StopRequested():
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
	if err := p.detector.ProcessPolicySync(p.getCurrentContext(), sync); err != nil {
		return errors.Wrap(err, "processing policy sync")
	}
	return nil
}

func (p *eventPipeline) processReprocessDeployments() error {
	log.Debug("ReprocessDeployments message received from central")
	if err := p.detector.ProcessReprocessDeployments(); err != nil {
		return errors.Wrap(err, "reprocessing deployments")
	}
	msg := component.NewEvent()
	// TODO(ROX-14310): Add WithSkipResolving to the DeploymentResolution (Revert: https://github.com/stackrox/stackrox/pull/5551)
	msg.AddDeploymentReference(resolver.ResolveAllDeployments(),
		component.WithForceDetection())
	msg.Context = p.getCurrentContext()
	p.resolver.Send(msg)
	return nil
}

func (p *eventPipeline) processUpdatedImage(image *storage.Image) error {
	log.Debugf("UpdatedImage message received from central: image name: %s, number of components: %d", image.GetName().GetFullName(), image.GetComponents())
	if err := p.detector.ProcessUpdatedImage(image); err != nil {
		return errors.Wrap(err, "updating image")
	}
	msg := component.NewEvent()
	msg.AddDeploymentReference(resolver.ResolveDeploymentsByImages(image),
		component.WithForceDetection(),
		component.WithSkipResolving())
	msg.Context = p.getCurrentContext()
	p.resolver.Send(msg)
	return nil
}

func (p *eventPipeline) processReprocessDeployment(req *central.ReprocessDeployment) error {
	log.Debug("ReprocessDeployment message received from central")
	if err := p.reprocessor.ProcessReprocessDeployments(req); err != nil {
		return errors.Wrap(err, "reprocessing deployment")
	}
	msg := component.NewEvent()
	msg.AddDeploymentReference(resolver.ResolveDeploymentIds(req.GetDeploymentIds()...),
		component.WithForceDetection(),
		component.WithSkipResolving())
	msg.Context = p.getCurrentContext()
	p.resolver.Send(msg)
	return nil
}

func (p *eventPipeline) processInvalidateImageCache(req *central.InvalidateImageCache) error {
	log.Debug("InvalidateImageCache message received from central")
	if err := p.reprocessor.ProcessInvalidateImageCache(req); err != nil {
		return errors.Wrap(err, "invalidating image cache")
	}
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
	return nil
}
