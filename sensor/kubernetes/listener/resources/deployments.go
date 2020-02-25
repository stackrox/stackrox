package resources

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/process/filter"
	"github.com/stackrox/rox/sensor/common/config"
	"github.com/stackrox/rox/sensor/common/detector"
	"github.com/stackrox/rox/sensor/kubernetes/listener/resources/references"
	v1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1listers "k8s.io/client-go/listers/core/v1"
)

// deploymentDispatcherImpl is a Dispatcher implementation for deployment events.
// All deploymentDispatcherImpl must share a handler instance since different types must be correlated.
type deploymentDispatcherImpl struct {
	deploymentType string

	handler *deploymentHandler
}

// newDeploymentDispatcher creates and returns a new deployment dispatcher instance.
func newDeploymentDispatcher(deploymentType string, handler *deploymentHandler) Dispatcher {
	return &deploymentDispatcherImpl{
		deploymentType: deploymentType,
		handler:        handler,
	}
}

// ProcessEvent processes a deployment resource events, and returns the sensor events to emit in response.
func (d *deploymentDispatcherImpl) ProcessEvent(obj, oldObj interface{}, action central.ResourceAction) []*central.SensorEvent {
	// Check owner references and build graph
	// Every single object should implement this interface
	metaObj, ok := obj.(metaV1.Object)
	if !ok {
		log.Errorf("could not process %+v as it does not implement metaV1.Object", obj)
		return nil
	}

	if action == central.ResourceAction_REMOVE_RESOURCE {
		d.handler.hierarchy.Remove(string(metaObj.GetUID()))
		return d.handler.processWithType(obj, oldObj, action, d.deploymentType)
	}

	parents := make([]string, 0, len(metaObj.GetOwnerReferences()))
	for _, ref := range metaObj.GetOwnerReferences() {
		if ref.UID != "" {
			parents = append(parents, string(ref.UID))
		}
	}
	d.handler.hierarchy.Add(parents, string(metaObj.GetUID()))
	return d.handler.processWithType(obj, oldObj, action, d.deploymentType)
}

// deploymentHandler handles deployment resource events and does the actual processing.
type deploymentHandler struct {
	podLister       v1listers.PodLister
	serviceStore    *serviceStore
	deploymentStore *DeploymentStore
	endpointManager *endpointManager
	namespaceStore  *namespaceStore
	processFilter   filter.Filter
	config          config.Handler
	hierarchy       references.ParentHierarchy
	rbac            rbacUpdater

	detector detector.Detector
}

// newDeploymentHandler creates and returns a new deployment handler.
func newDeploymentHandler(serviceStore *serviceStore, deploymentStore *DeploymentStore, endpointManager *endpointManager, namespaceStore *namespaceStore,
	rbac rbacUpdater, podLister v1listers.PodLister, processFilter filter.Filter, config config.Handler, detector detector.Detector) *deploymentHandler {
	return &deploymentHandler{
		podLister:       podLister,
		serviceStore:    serviceStore,
		deploymentStore: deploymentStore,
		endpointManager: endpointManager,
		namespaceStore:  namespaceStore,
		processFilter:   processFilter,
		config:          config,
		hierarchy:       references.NewParentHierarchy(),
		rbac:            rbac,

		detector: detector,
	}
}

func (d *deploymentHandler) processWithType(obj, oldObj interface{}, action central.ResourceAction, deploymentType string) []*central.SensorEvent {
	wrap := newDeploymentEventFromResource(obj, &action, deploymentType, d.podLister, d.namespaceStore, d.hierarchy, d.config.GetConfig().GetRegistryOverride())
	if wrap == nil {
		return d.maybeProcessPod(obj, oldObj, action)
	}
	wrap.updatePortExposureFromStore(d.serviceStore)
	if action != central.ResourceAction_REMOVE_RESOURCE {
		d.deploymentStore.addOrUpdateDeployment(wrap)
		d.endpointManager.OnDeploymentCreateOrUpdate(wrap)
		d.processFilter.Update(wrap.GetDeployment())
		d.rbac.assignPermissionLevelToDeployment(wrap)
	} else {
		d.deploymentStore.removeDeployment(wrap)
		d.endpointManager.OnDeploymentRemove(wrap)
		d.processFilter.Delete(wrap.GetId())
	}
	if features.SensorBasedDetection.Enabled() {
		d.detector.ProcessDeployment(wrap.GetDeployment(), action)
	}
	return []*central.SensorEvent{wrap.toEvent(action)}
}

func (d *deploymentHandler) maybeProcessPod(obj, oldObj interface{}, action central.ResourceAction) []*central.SensorEvent {
	pod, ok := obj.(*v1.Pod)
	if !ok {
		return nil
	}

	// We care if the pod is running OR if the pod is being removed as that can impact the top level object
	if pod.Status.Phase != v1.PodRunning && action != central.ResourceAction_REMOVE_RESOURCE {
		return nil
	}

	if action != central.ResourceAction_REMOVE_RESOURCE && oldObj != nil {
		oldPod, ok := oldObj.(*v1.Pod)
		if !ok {
			log.Error("previous version of pod is not a pod")
			return nil
		}
		// We care when pods are transitioning to running so ensure that the old pod status is not RUNNING
		// In the cases of CREATES or UPDATES
		if oldPod.Status.Phase == v1.PodRunning {
			return nil
		}
	}

	owners := d.deploymentStore.getOwningDeployments(pod.Namespace, pod.Labels)
	var events []*central.SensorEvent
	for _, owner := range owners {
		events = append(events, d.processWithType(owner.original, nil, central.ResourceAction_UPDATE_RESOURCE, owner.Type)...)
	}
	return events
}
