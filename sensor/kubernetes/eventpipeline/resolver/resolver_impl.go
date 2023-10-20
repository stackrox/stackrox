package resolver

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/sensor/common/metrics"
	"github.com/stackrox/rox/sensor/common/store"
	"github.com/stackrox/rox/sensor/kubernetes/eventpipeline/component"
)

var (
	log = logging.LoggerForModule()
)

type resolverImpl struct {
	outputQueue component.OutputQueue
	innerQueue  chan *component.ResourceEvent

	storeProvider store.Provider
	stopSig       concurrency.Signal
}

// Start the resolverImpl component
func (r *resolverImpl) Start() error {
	go r.runResolver()
	return nil
}

// Stop the resolverImpl component
func (r *resolverImpl) Stop(_ error) {
	r.stopSig.Signal()
}

// Send a ResourceEvent message to the inner queue
func (r *resolverImpl) Send(event *component.ResourceEvent) {
	r.innerQueue <- event
	metrics.IncResolverChannelSize()
}

// runResolver reads messages from the inner queue and process the message
func (r *resolverImpl) runResolver() {
	for {
		select {
		case msg, more := <-r.innerQueue:
			if !more {
				return
			}
			r.processMessage(msg)
			metrics.DecResolverChannelSize()
		case <-r.stopSig.Done():
			return
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

			referenceIds := deploymentReference.Reference(r.storeProvider.Deployments())

			if deploymentReference.ForceDetection && len(referenceIds) > 0 {
				// We append the referenceIds to the msg to be reprocessed
				msg.AddDeploymentForReprocessing(referenceIds...)
			}

			for _, id := range referenceIds {
				preBuiltDeployment := r.storeProvider.Deployments().Get(id)
				if preBuiltDeployment == nil {
					log.Warnf("Deployment with id %s not found", id)
					continue
				}
				// Skip resolving the deployment dependencies
				if deploymentReference.SkipResolving {
					d, built := r.storeProvider.Deployments().GetBuiltDeployment(id)
					if d == nil {
						log.Warnf("Deployment with id %s not found", id)
						continue
					}

					if !built {
						log.Debugf("Deployment with id %s is already in the pipeline, skipping processing", id)
						continue
					}
					msg.AddDeploymentForDetection(component.DetectorMessage{Object: d, Action: deploymentReference.ParentResourceAction})
					continue
				}

				// Remove actions are done at the handler level. This is not ideal but for now it allows us to be able to fetch deployments from the store
				// in the resolver instead of sending a copy. We still manage OnDeploymentCreateOrUpdate here.
				r.storeProvider.EndpointManager().OnDeploymentCreateOrUpdateByID(id)

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

				d, newObject, err := r.storeProvider.Deployments().BuildDeploymentWithDependencies(id, store.Dependencies{
					PermissionLevel: permissionLevel,
					Exposures:       exposureInfo,
					LocalImages:     localImages,
				})

				if err != nil {
					log.Warnf("Failed to build deployment dependency: %s", err)
					continue
				}

				// Skip generating an event and sending the deployment to the detector if the object is not
				// new and detection isn't forced.
				if !deploymentReference.ForceDetection && newObject {
					msg.AddSensorEvent(toEvent(deploymentReference.ParentResourceAction, d, msg.DeploymentTiming)).
						AddDeploymentForDetection(component.DetectorMessage{Object: d, Action: deploymentReference.ParentResourceAction})
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
			Deployment: deployment.Clone(),
		},
	}
}

var _ component.Resolver = (*resolverImpl)(nil)
