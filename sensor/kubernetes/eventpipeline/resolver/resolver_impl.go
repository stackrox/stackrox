package resolver

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
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
}

// Start the resolverImpl component
func (r *resolverImpl) Start() error {
	go r.runResolver()
	return nil
}

// Stop the resolverImpl component
func (r *resolverImpl) Stop(_ error) {
	defer close(r.innerQueue)
}

// Send a ResourceEvent message to the inner queue
func (r *resolverImpl) Send(event *component.ResourceEvent) {
	r.innerQueue <- event
}

// runResolver reads messages from the inner queue and process the message
func (r *resolverImpl) runResolver() {
	for {
		msg, more := <-r.innerQueue
		if !more {
			return
		}
		r.processMessage(msg)
	}
}

// processMessage resolves the dependencies and forwards the message to the outputQueue
func (r *resolverImpl) processMessage(msg *component.ResourceEvent) {
	if msg.DeploymentReference != nil {
		referenceIds := msg.DeploymentReference(r.storeProvider.Deployments())

		if msg.ForceDetection && len(referenceIds) > 0 {
			// We append the referenceIds to the msg to be reprocessed
			msg = component.MergeResourceEvents(msg, component.NewResourceEvent(nil, nil, referenceIds))
		}

		for _, id := range referenceIds {
			preBuiltDeployment := r.storeProvider.Deployments().Get(id)
			if preBuiltDeployment == nil {
				log.Warnf("Deployment with id %s not found", id)
				continue
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
				continue
			}

			event := component.NewResourceEvent([]*central.SensorEvent{toEvent(msg.ParentResourceAction, d, msg.DeploymentTiming)},
				[]component.CompatibilityDetectionMessage{{Object: d, Action: msg.ParentResourceAction}}, nil)

			component.MergeResourceEvents(msg, event)
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
