package resources

import (
	"sort"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/process/filter"
	"github.com/stackrox/rox/pkg/protoconv/resources"
	"github.com/stackrox/rox/sensor/common/clusterid"
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
		if ref.UID != "" && resources.IsTrackedOwnerReference(ref) {
			// Only bother adding parents we track.
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
	podStore        *podStore
	endpointManager *endpointManager
	namespaceStore  *namespaceStore
	processFilter   filter.Filter
	config          config.Handler
	hierarchy       references.ParentHierarchy
	rbac            rbacUpdater

	detector detector.Detector
}

// newDeploymentHandler creates and returns a new deployment handler.
func newDeploymentHandler(serviceStore *serviceStore, deploymentStore *DeploymentStore, podStore *podStore,
	endpointManager *endpointManager, namespaceStore *namespaceStore, rbac rbacUpdater, podLister v1listers.PodLister,
	processFilter filter.Filter, config config.Handler, detector detector.Detector) *deploymentHandler {
	return &deploymentHandler{
		podLister:       podLister,
		serviceStore:    serviceStore,
		deploymentStore: deploymentStore,
		podStore:        podStore,
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
	var events []*central.SensorEvent
	if pod, ok := obj.(*v1.Pod); !ok && wrap == nil {
		// This is not a tracked resource nor a pod.
		return nil
	} else if ok {
		// This is a pod. It may or may not be tracked.
		if wrap == nil {
			// This pod is not a top-level object that we track.
			// Call maybeProcessPodEvent because we may need to update this pod's respective top-level resources.
			return d.maybeProcessPodEvent(pod, oldObj, action)
		}
		if features.PodDeploymentSeparation.Enabled() {
			// This pod is a top-level resource that we track.
			events = append(events, d.processPodEvent(wrap, pod, action))
		}
	}

	wrap.ClusterId = clusterid.Get()
	wrap.updatePortExposureFromStore(d.serviceStore)
	if action != central.ResourceAction_REMOVE_RESOURCE {
		d.deploymentStore.addOrUpdateDeployment(wrap)
		d.endpointManager.OnDeploymentCreateOrUpdate(wrap)
		if !features.PodDeploymentSeparation.Enabled() {
			d.processFilter.Update(wrap.GetDeployment())
		}
		d.rbac.assignPermissionLevelToDeployment(wrap)
	} else {
		d.deploymentStore.removeDeployment(wrap)
		if features.PodDeploymentSeparation.Enabled() {
			d.podStore.onDeploymentRemove(wrap)
		}
		d.endpointManager.OnDeploymentRemove(wrap)
		d.processFilter.Delete(wrap.GetId())
	}
	d.detector.ProcessDeployment(wrap.GetDeployment(), action)
	events = append(events, wrap.toEvent(action))
	return events
}

// maybeProcessPodEvent may return SensorEvents indicating a change in a deployment's state based on updated pod state.
// If PodDeploymentSeparation is enabled, then it will definitely return at least a SensorEvent indicating
// a change in the pod's state iff it can associate the pod back to a single top-level resource.
func (d *deploymentHandler) maybeProcessPodEvent(pod *v1.Pod, oldObj interface{}, action central.ResourceAction) []*central.SensorEvent {
	// Hierarchy only tracks process a process's parents if they are resources that we track as a Deployment.
	// We also only track top-level objects (ex we track Deployment resources in favor of the underlying ReplicaSet and Pods)
	// as our version of a Deployment, so the only parents we'd want to potentially process are the top-level ones.
	owners := d.deploymentStore.getDeploymentsByIDs(pod.Namespace, d.hierarchy.TopLevelParents(string(pod.GetUID())))
	var events []*central.SensorEvent
	if features.PodDeploymentSeparation.Enabled() {
		if len(owners) != 1 {
			var candidates []string
			for _, candidate := range owners {
				candidates = append(candidates, candidate.GetId())
			}
			log.Errorf("cannot associate the pod %s/%s back to a single deployment wrapper; candidates: %+v", pod.GetNamespace(), pod.GetName(), candidates)
			return nil
		}
		log.Debugf("Owner of %s is %s", pod.Name, owners[0].Name)
		events = append(events, d.processPodEvent(owners[0], pod, action))
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
	for _, owner := range owners {
		events = append(events, d.processWithType(owner.original, nil, central.ResourceAction_UPDATE_RESOURCE, owner.Type)...)
	}
	return events
}

// processPodEvent returns a SensorEvent indicating a change in a pod's state.
func (d *deploymentHandler) processPodEvent(wrap *deploymentWrap, pod *v1.Pod, action central.ResourceAction) *central.SensorEvent {
	// TODO: This is called after some prior work potentially changes the action.
	// If this pod were also the top-level deployment, then if it's status is SUCCEEDED or FAILED, then we set the
	// action to REMOVE_RESOURCE before this point. See if this matters...
	p := &storage.Pod{
		Id:           string(pod.GetUID()),
		DeploymentId: wrap.GetId(),
		ClusterId:    wrap.GetClusterId(),
		Namespace:    wrap.GetNamespace(),
		Active:       action != central.ResourceAction_REMOVE_RESOURCE,
	}
	for i, instance := range containerInstances(pod) {
		// This check that the size is not greater is necessary, because pods can be in terminating as a deployment is updated
		// The deployment will still be managing the pods, but we want to take the new pod(s) as the source of truth
		if i >= len(wrap.GetContainers()) {
			break
		}
		p.Instances = append(p.Instances, instance)
	}
	// Create a stable ordering
	sort.SliceStable(p.Instances, func(i, j int) bool { return p.Instances[i].InstanceId.Id < p.Instances[j].InstanceId.Id })

	if action == central.ResourceAction_REMOVE_RESOURCE {
		d.podStore.removePod(p)
	} else {
		d.podStore.addOrUpdatePod(p)
		d.processFilter.UpdateByGivenContainers(p.DeploymentId, d.podStore.getContainersForDeployment(p.Namespace, p.DeploymentId))
	}

	log.Debugf("Action: %+v Pod: %+v", action, p)

	return &central.SensorEvent{
		Id:     p.GetId(),
		Action: action,
		Resource: &central.SensorEvent_Pod{
			Pod: p,
		},
	}
}
