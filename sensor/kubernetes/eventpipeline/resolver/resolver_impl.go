package resolver

import (
	"sync/atomic"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/uniqueue"
	"github.com/stackrox/rox/sensor/common/metrics"
	"github.com/stackrox/rox/sensor/common/store"
	"github.com/stackrox/rox/sensor/kubernetes/eventpipeline/component"
)

var (
	log = logging.LoggerForModule()
)

type deploymentRef struct {
	id     string
	skip   bool
	action central.ResourceAction
}

type resolverImpl struct {
	stopper        concurrency.Stopper
	outputQueue    component.OutputQueue
	innerQueue     chan *component.ResourceEvent
	innerQueueSize int

	storeProvider store.Provider
	stopped       *atomic.Bool
	queue         *uniqueue.UniQueue[deploymentRef]
}

// Start the resolverImpl component
func (r *resolverImpl) Start() error {
	if err := r.queue.Start(); err != nil {
		return err
	}
	r.stopper.LowLevel().ResetStopRequest()
	r.innerQueue = make(chan *component.ResourceEvent, r.innerQueueSize)
	go r.runResolver()
	return nil
}

// Stop the resolverImpl component
func (r *resolverImpl) Stop(_ error) {
	if !r.stopper.Client().Stopped().IsDone() {
		defer func() {
			_ = r.stopper.Client().Stopped().Wait()
			r.queue.Stop()
			close(r.innerQueue)
			r.stopped.Store(true)
		}()
	}
	r.stopper.Client().Stop()
}

// Send a ResourceEvent message to the inner queue
func (r *resolverImpl) Send(event *component.ResourceEvent) {
	if !r.stopped.Load() {
		r.innerQueue <- event
		metrics.IncResolverChannelSize()
	}
}

// runResolver reads messages from the inner queue and process the message
func (r *resolverImpl) runResolver() {
	defer r.stopper.Flow().ReportStopped()
	wg := &sync.WaitGroup{}
	wg.Add(2)
	go r.runSingleDeploymentResolver(wg)
	go r.runProcessMessages(wg)
	wg.Wait()
}

func (r *resolverImpl) runProcessMessages(wg *sync.WaitGroup) {
	defer wg.Done()
	for {
		select {
		case <-r.stopper.Flow().StopRequested():
			return
		case msg, more := <-r.innerQueue:
			if !more {
				return
			}
			r.processMessage(msg)
			metrics.DecResolverChannelSize()
		}
	}
}

func (r *resolverImpl) pushDeploymentIDs(skip bool, action central.ResourceAction, ids ...string) {
	for _, id := range ids {
		r.queue.PushC() <- deploymentRef{id, skip, action}
	}
}

func (r *resolverImpl) runSingleDeploymentResolver(wg *sync.WaitGroup) {
	defer wg.Done()
	for {
		select {
		case <-r.stopper.Flow().StopRequested():
			return
		case msg, more := <-r.queue.PopC():
			if !more {
				return
			}
			r.resolveDeployment(msg.id, msg.skip, msg.action)
		}
	}
}

func (r *resolverImpl) resolveDeployment(id string, skip bool, action central.ResourceAction) {
	msg := component.NewEvent()
	if skip {
		d, built := r.storeProvider.Deployments().GetBuiltDeployment(id)
		if d == nil {
			log.Warnf("Deployment with id %s not found", id)
			return
		}

		if !built {
			log.Debugf("Deployment with id %s is already in the pipeline, skipping processing", id)
			return
		}
		msg.AddDeploymentForDetection(component.DetectorMessage{Object: d, Action: action})
		r.outputQueue.Send(msg)
		return
	}
	preBuiltDeployment := r.storeProvider.Deployments().Get(id)
	if preBuiltDeployment == nil {
		log.Warnf("Deployment with id %s not found", id)
		return
	}
	// Remove actions are done at the handler level. This is not ideal but for now it allows us to be able to fetch deployments from the store
	// in the resolver instead of sending a copy. We still manage OnDeploymentCreateOrUpdate here.
	r.storeProvider.EndpointManager().OnDeploymentCreateOrUpdateByID(id)

	permissionLevel := r.storeProvider.RBAC().GetPermissionLevelForDeployment(preBuiltDeployment)
	exposureInfo := r.storeProvider.Services().
		GetExposureInfos(preBuiltDeployment.GetNamespace(), preBuiltDeployment.GetPodLabels())

	d, err := r.storeProvider.Deployments().BuildDeploymentWithDependencies(id, store.Dependencies{
		PermissionLevel: permissionLevel,
		Exposures:       exposureInfo,
	})

	if err != nil {
		log.Warnf("Failed to build deployment dependency: %s", err)
		return
	}

	msg.AddSensorEvent(toEvent(action, d, msg.DeploymentTiming)).
		AddDeploymentForDetection(component.DetectorMessage{Object: d, Action: action})
	r.outputQueue.Send(msg)
}

// processMessage resolves the dependencies and forwards the message to the outputQueue
func (r *resolverImpl) processMessage(msg *component.ResourceEvent) {
	if msg.DeploymentReferences != nil {

		for _, deploymentReference := range msg.DeploymentReferences {
			if deploymentReference.Reference == nil {
				continue
			}

			referenceIds := deploymentReference.Reference(r.storeProvider.Deployments())

			if deploymentReference.ForceDetection && len(referenceIds) > 0 {
				// We append the referenceIds to the msg to be reprocessed
				msg.AddDeploymentForReprocessing(referenceIds...)
				for _, id := range referenceIds {
					r.resolveDeployment(id, deploymentReference.SkipResolving, deploymentReference.ParentResourceAction)
				}
				continue
			}

			r.pushDeploymentIDs(deploymentReference.SkipResolving, deploymentReference.ParentResourceAction, referenceIds...)
		}

	}

	r.outputQueue.Send(msg)
}

func toEvent(action central.ResourceAction, deployment *storage.Deployment, timing *central.Timing) *central.SensorEvent {
	return &central.SensorEvent{
		Id:     deployment.GetId(),
		Action: action,
		Timing: timing,
		Resource: &central.SensorEvent_Deployment{
			Deployment: deployment.Clone(),
		},
	}
}

var _ component.Resolver = (*resolverImpl)(nil)
