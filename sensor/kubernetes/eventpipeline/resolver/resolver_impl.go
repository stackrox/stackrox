package resolver

import (
	"context"
	"fmt"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/dedupingqueue"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/sensor/common/metrics"
	"github.com/stackrox/rox/sensor/common/store"
	"github.com/stackrox/rox/sensor/kubernetes/eventpipeline/component"
)

var (
	log = logging.LoggerForModule()
)

type deploymentRef struct {
	context          context.Context
	id               string
	action           central.ResourceAction
	forceDetection   bool
	skipResolving    bool
	deploymentTiming *central.Timing
}

// GetDedupeKey returns the key to index the deploymentRef in the queue
func (d *deploymentRef) GetDedupeKey() string {
	return fmt.Sprintf("%s-%s-%t-%t", d.id, d.action.String(), d.skipResolving, d.forceDetection)
}

type resolverImpl struct {
	outputQueue component.OutputQueue
	innerQueue  chan *component.ResourceEvent

	storeProvider store.Provider
	stopper       concurrency.Stopper

	deploymentRefQueue *dedupingqueue.DedupingQueue[string]
}

// Start the resolverImpl component
func (r *resolverImpl) Start() error {
	go r.runResolver()
	if features.SensorAggregateDeploymentReferenceOptimization.Enabled() && r.deploymentRefQueue != nil {
		go r.runPullAndResolve()
	}
	return nil
}

// Stop the resolverImpl component
func (r *resolverImpl) Stop() {
	if !r.stopper.Client().Stopped().IsDone() {
		defer func() {
			_ = r.stopper.Client().Stopped().Wait()
		}()
	}
	r.stopper.Client().Stop()
}

// Send a ResourceEvent message to the inner queue
func (r *resolverImpl) Send(event *component.ResourceEvent) {
	r.innerQueue <- event
	metrics.IncResolverChannelSize()
}

// runResolver reads messages from the inner queue and process the message
func (r *resolverImpl) runResolver() {
	defer r.stopper.Flow().ReportStopped()
	for {
		select {
		case msg, more := <-r.innerQueue:
			if !more {
				return
			}
			r.processMessage(msg)
			metrics.DecResolverChannelSize()
		case <-r.stopper.Flow().StopRequested():
			return
		}
	}
}

// resolveDeployment resolves the given deployment reference. Returns false if the deployment was not resolved, true otherwise.
func (r *resolverImpl) resolveDeployment(msg *component.ResourceEvent, ref *deploymentRef) bool {
	if ref.forceDetection {
		// We append the referenceIds to the msg to be reprocessed
		msg.AddDeploymentForReprocessing(ref.id)
	}
	preBuiltDeployment := r.storeProvider.Deployments().Get(ref.id)
	if preBuiltDeployment == nil {
		log.Warnf("Deployment with id %s not found", ref.id)
		return false
	}
	// Skip resolving the deployment dependencies
	if ref.skipResolving {
		d, built := r.storeProvider.Deployments().GetBuiltDeployment(ref.id)
		if d == nil {
			log.Warnf("Deployment with id %s not found", ref.id)
			return false
		}

		if !built {
			log.Debugf("Deployment with id %s is already in the pipeline, skipping processing", ref.id)
			return false
		}
		msg.AddDeploymentForDetection(component.DeploytimeDetectionRequest{Object: d, Action: ref.action})
		return true
	}

	// Remove actions are done at the handler level. This is not ideal but for now it allows us to be able to fetch deployments from the store
	// in the resolver instead of sending a copy. We still manage OnDeploymentCreateOrUpdateByID here.
	r.storeProvider.EndpointManager().OnDeploymentCreateOrUpdateByID(ref.id)

	localImages := set.NewStringSet()
	for _, c := range preBuiltDeployment.GetContainers() {
		imgName := c.GetImage().GetName()
		if r.storeProvider.Registries().IsLocal(imgName) {
			localImages.Add(imgName.GetFullName())
		}
	}

	permissionLevel := r.storeProvider.RBAC().GetPermissionLevelForDeployment(preBuiltDeployment)
	exposureInfo := r.storeProvider.Services().
		GetExposureInfos(preBuiltDeployment.GetNamespace(), preBuiltDeployment.GetPodLabels())

	d, newObject, err := r.storeProvider.Deployments().BuildDeploymentWithDependencies(ref.id, store.Dependencies{
		PermissionLevel: permissionLevel,
		Exposures:       exposureInfo,
		LocalImages:     localImages,
	})

	if err != nil {
		log.Warnf("Failed to build deployment dependency: %s", err)
		return false
	}

	// Skip generating an event and sending the deployment to the detector if the object is not
	// new and detection isn't forced.
	if ref.forceDetection || newObject {
		msg.AddSensorEvent(toEvent(ref.action, d, msg.DeploymentTiming)).
			AddDeploymentForDetection(component.DeploytimeDetectionRequest{Object: d, Action: ref.action})
		return true
	}
	return false
}

// runPullAndResolve pull the next deployment reference to be resolved out of the queue
func (r *resolverImpl) runPullAndResolve() {
	for {
		item := r.deploymentRefQueue.PullBlocking(r.stopper.LowLevel().GetStopRequestSignal())
		select {
		case <-r.stopper.Flow().StopRequested():
			return
		default:
		}
		ref, ok := item.(*deploymentRef)
		if !ok {
			log.Warnf("The pulled item is not a deploymentRef")
			continue
		}

		if ref == nil {
			continue
		}
		msg := component.NewEvent()
		msg.Context = ref.context
		msg.DeploymentTiming = ref.deploymentTiming
		if r.resolveDeployment(msg, ref) {
			r.outputQueue.Send(msg)
		}
	}
}

// processMessage resolves the dependencies and forwards the message to the outputQueue
func (r *resolverImpl) processMessage(msg *component.ResourceEvent) {
	if msg.DeploymentReferences != nil {

		for _, deploymentReference := range msg.DeploymentReferences {
			if deploymentReference.Reference == nil {
				continue
			}

			// Runs the deployment resolution. This callback will fetch all the deployment IDs that are affected by
			// the resource event.
			referenceIds := deploymentReference.Reference(r.storeProvider.Deployments())

			for _, id := range referenceIds {
				ref := &deploymentRef{
					context:          msg.Context,
					deploymentTiming: msg.DeploymentTiming,
					id:               id,
					action:           deploymentReference.ParentResourceAction,
					skipResolving:    deploymentReference.SkipResolving,
					forceDetection:   deploymentReference.ForceDetection,
				}
				if features.SensorAggregateDeploymentReferenceOptimization.Enabled() && r.deploymentRefQueue != nil {
					r.deploymentRefQueue.Push(ref)
				} else {
					r.resolveDeployment(msg, ref)
				}
			}
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
			Deployment: deployment.CloneVT(),
		},
	}
}

var _ component.Resolver = (*resolverImpl)(nil)
