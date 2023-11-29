package resources

import (
	"sort"

	"github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/process/filter"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stackrox/rox/sensor/common/awscredentials"
	"github.com/stackrox/rox/sensor/common/config"
	"github.com/stackrox/rox/sensor/common/registry"
	"github.com/stackrox/rox/sensor/common/store"
	"github.com/stackrox/rox/sensor/common/store/resolver"
	"github.com/stackrox/rox/sensor/kubernetes/eventpipeline/component"
	"github.com/stackrox/rox/sensor/kubernetes/listener/resources/rbac"
	"github.com/stackrox/rox/sensor/kubernetes/listener/resources/references"
	"github.com/stackrox/rox/sensor/kubernetes/orchestratornamespaces"
	v1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1listers "k8s.io/client-go/listers/core/v1"
)

var (
	// It is highly recommended that nobody change this value unless they are absolutely sure,
	// but even then maybe don't do it.
	podNamespace = uuid.FromStringOrPanic("32581326-b68f-49f5-a8a2-83853cac8813")
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
func (d *deploymentDispatcherImpl) ProcessEvent(obj, oldObj interface{}, action central.ResourceAction) *component.ResourceEvent {
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
	d.handler.hierarchy.Add(metaObj)
	return d.handler.processWithType(obj, oldObj, action, d.deploymentType)
}

// deploymentHandler handles deployment resource events and does the actual processing.
type deploymentHandler struct {
	podLister              v1listers.PodLister
	serviceStore           store.ServiceStore
	deploymentStore        *DeploymentStore
	podStore               *PodStore
	endpointManager        endpointManager
	namespaceStore         *namespaceStore
	processFilter          filter.Filter
	config                 config.Handler
	credentialsManager     awscredentials.RegistryCredentialsManager
	hierarchy              references.ParentHierarchy
	rbac                   rbac.Store
	orchestratorNamespaces *orchestratornamespaces.OrchestratorNamespaces
	registryStore          *registry.Store

	clusterID string
}

// newDeploymentHandler creates and returns a new deployment handler.
func newDeploymentHandler(
	clusterID string,
	serviceStore store.ServiceStore,
	deploymentStore *DeploymentStore,
	podStore *PodStore,
	endpointManager endpointManager,
	namespaceStore *namespaceStore,
	rbac rbac.Store,
	podLister v1listers.PodLister,
	processFilter filter.Filter,
	config config.Handler,
	namespaces *orchestratornamespaces.OrchestratorNamespaces,
	registryStore *registry.Store,
	credentialsManager awscredentials.RegistryCredentialsManager,
) *deploymentHandler {
	return &deploymentHandler{
		podLister:              podLister,
		serviceStore:           serviceStore,
		deploymentStore:        deploymentStore,
		podStore:               podStore,
		endpointManager:        endpointManager,
		namespaceStore:         namespaceStore,
		processFilter:          processFilter,
		config:                 config,
		hierarchy:              references.NewParentHierarchy(),
		rbac:                   rbac,
		orchestratorNamespaces: namespaces,
		registryStore:          registryStore,
		clusterID:              clusterID,
		credentialsManager:     credentialsManager,
	}
}

func (d *deploymentHandler) processWithType(obj, oldObj interface{}, action central.ResourceAction, deploymentType string) *component.ResourceEvent {
	deploymentWrap := newDeploymentEventFromResource(obj, &action, deploymentType, d.clusterID, d.podLister, d.namespaceStore,
		d.hierarchy, d.config.GetConfig().GetRegistryOverride(), d.orchestratorNamespaces)
	// Note: deploymentWrap may be nil. Typically, this means that this is not a top-level object that we track --
	// either it's an object we don't track, or we track its parent.
	// (For example, we don't track replicasets if they are owned by a deployment.)
	// We don't immediately return if deploymentWrap == nil though,
	// because IF the object is a pod, we want to process the pod event.
	objAsPod, _ := obj.(*v1.Pod)

	events := &component.ResourceEvent{
		ForwardMessages:      []*central.SensorEvent{},
		DetectorMessages:     []component.DetectorMessage{},
		ReprocessDeployments: []string{},
		DeploymentTiming:     nil,
		DeploymentReferences: []component.DeploymentReference{},
	}
	// If the object is a pod, process the pod event.
	if objAsPod != nil {
		var owningDeploymentID string
		uid := string(objAsPod.GetUID())
		if deploymentWrap != nil && deploymentWrap.GetType() != k8sStandalonePodType {
			// The pod is a top-level object, so it is its own owner.
			owningDeploymentID = uid
		} else {
			// Fetch the owning deploymentIDs from the hierarchy.
			owningDeploymentIDs := d.hierarchy.TopLevelParents(uid)
			switch owningDeploymentIDs.Cardinality() {
			case 0:
				// See comment below the if-else about why we don't log on removes.
				if action != central.ResourceAction_REMOVE_RESOURCE {
					log.Warnf("Found no owners for pod %s (%s/%s)", uid, objAsPod.Namespace, objAsPod.Name)
				}
			case 1:
				owningDeploymentID = owningDeploymentIDs.GetArbitraryElem()
			default:
				log.Warnf("Found multiple owners (%v) for pod %s (%s/%s). Dropping the pod update...",
					owningDeploymentIDs.AsSlice(), uid, objAsPod.Namespace, objAsPod.Name)
			}
		}

		// On removes, we may not get the owning deployment ID if the deployment was deleted before the pod.
		// This is okay. We still want to send the remove event anyway.
		if action == central.ResourceAction_REMOVE_RESOURCE || owningDeploymentID != "" {
			removeEvents := d.processPodEvent(owningDeploymentID, objAsPod, action)
			if removeEvents != nil {
				events.AddSensorEvent(removeEvents)
			}
		}

		if deploymentWrap == nil {
			// It's only a pod event and the pod belongs to another resource (e.g. deployment)
			d.maybeUpdateParentsOfPod(objAsPod, oldObj, action, events)
			return events
		}
	}

	// If it's an object we don't generate an event for (e.g. ReplicaSets) processing should stop here.
	if deploymentWrap == nil {
		return events
	}

	events = d.appendIntegrationsOnCredentials(action, deploymentWrap.GetContainers(), events)

	if action == central.ResourceAction_REMOVE_RESOURCE {
		// TODO(ROX-14309): move this logic to the resolver
		// We need to do this here since the resolver relies on the deploymentStore to have the wrap
		d.endpointManager.OnDeploymentRemove(deploymentWrap)
		// At the moment we need to also send this deployment to the compatibility module when it's being deleted.
		// Moving forward, there might be a different way to solve this, for example by changing the compatibility
		// module to accept only deployment IDs rather than the entire deployment object. For more info on this
		// check the PR comment here: https://github.com/stackrox/stackrox/pull/3695#discussion_r1030214615
		events.AddDeploymentForDetection(component.DetectorMessage{
			Object: deploymentWrap.GetDeployment(),
			Action: action,
		}).AddSensorEvent(deploymentWrap.toEvent(action)) // if resource is being removed, we can create the remove message here without related resources
	} else {
		// If re-sync is disabled, we don't need to process deployment relationships here. We pass a deployment
		// references up the chain, which will be used to trigger the actual deployment event and detection.
		events.AddDeploymentReference(resolver.ResolveDeploymentIds(deploymentWrap.GetId()),
			component.WithParentResourceAction(action))
	}

	// Upsert/Delete at the end to avoid data race with other dispatchers
	if action != central.ResourceAction_REMOVE_RESOURCE {
		d.deploymentStore.addOrUpdateDeployment(deploymentWrap)
	} else {
		d.deploymentStore.removeDeployment(deploymentWrap)
		d.podStore.onDeploymentRemove(deploymentWrap)
		d.processFilter.Delete(deploymentWrap.GetId())
	}

	return events
}

// appendIntegrationsOnCredentials if credentials are found for registries used
// in the deployment, emit Registry Integration events for them.
//
// The method doesn't process REMOVE_RESOURCE actions. Notice that this means
// integrations are only recreated if the deployment exists, so it can be
// permanently deleted in Central.
func (d *deploymentHandler) appendIntegrationsOnCredentials(
	action central.ResourceAction,
	containers []*storage.Container,
	events *component.ResourceEvent,
) *component.ResourceEvent {
	if d.credentialsManager == nil || action == central.ResourceAction_REMOVE_RESOURCE {
		return events
	}
	registries := set.NewStringSet()
	for _, c := range containers {
		if r := c.GetImage().GetName().GetRegistry(); registries.Add(r) {
			if e := d.getImageIntegrationEvent(r); e != nil {
				events.AddSensorEvent(e)
			}
		}
	}
	return events
}

func (d *deploymentHandler) getImageIntegrationEvent(registry string) *central.SensorEvent {
	credentials := d.credentialsManager.GetRegistryCredentials(registry)
	if credentials == nil {
		return nil
	}
	expiresAt, err := types.TimestampProto(credentials.ExpirestAt)
	if err != nil {
		log.Error("ignoring invalid registry credentials: failed to parse timestamp")
		return nil
	}
	// Currently, all AWS registry credentials are handled as ECR image integrations, hence
	// type = "ecr".
	return &central.SensorEvent{
		Action: central.ResourceAction_UPDATE_RESOURCE,
		Resource: &central.SensorEvent_ImageIntegration{
			ImageIntegration: &storage.ImageIntegration{
				Id:         uuid.NewV4().String(),
				Type:       "ecr",
				Categories: []storage.ImageIntegrationCategory{storage.ImageIntegrationCategory_REGISTRY},
				IntegrationConfig: &storage.ImageIntegration_Ecr{
					Ecr: &storage.ECRConfig{
						Region:     credentials.AWSRegion,
						RegistryId: credentials.AWSAccount,
						AuthorizationData: &storage.ECRConfig_AuthorizationData{
							ExpiresAt: expiresAt,
							Username:  credentials.DockerConfig.Username,
							Password:  credentials.DockerConfig.Password,
						},
					},
				},
				Autogenerated: true,
			},
		},
	}
}

// maybeUpdateParentsOfPod may return SensorEvents indicating a change in a deployment's state based on updated pod state.
// We do this to ensure that the image IDs in the deployment are updated based on the actual running images in the pod.
func (d *deploymentHandler) maybeUpdateParentsOfPod(pod *v1.Pod, oldObj interface{}, action central.ResourceAction, rootEvent *component.ResourceEvent) {
	// We care if the pod is running OR if the pod is being removed as that can impact the top level object
	if pod.Status.Phase != v1.PodRunning && action != central.ResourceAction_REMOVE_RESOURCE {
		return
	}

	if action != central.ResourceAction_REMOVE_RESOURCE && oldObj != nil {
		oldPod, ok := oldObj.(*v1.Pod)
		if !ok {
			utils.Should(errors.Errorf("previous version of pod is not a pod (got %T)", oldObj))
			return
		}
		// We care when pods are transitioning to running so ensure that the old pod status is not RUNNING
		// In the cases of CREATES or UPDATES
		if oldPod.Status.Phase == v1.PodRunning {
			return
		}
	}

	// Hierarchy only tracks process a process's parents if they are resources that we track as a Deployment.
	// We also only track top-level objects (ex we track Deployment resources in favor of the underlying ReplicaSet and Pods)
	// as our version of a Deployment, so the only parents we'd want to potentially process are the top-level ones.
	owners := d.deploymentStore.getDeploymentsByIDs(pod.Namespace, d.hierarchy.TopLevelParents(string(pod.GetUID())))
	for _, owner := range owners {
		ev := d.processWithType(owner.original, nil, central.ResourceAction_UPDATE_RESOURCE, owner.Type)
		rootEvent.MergeResourceEvent(ev)
	}
}

// processPodEvent returns a SensorEvent indicating a change in a pod's state.
func (d *deploymentHandler) processPodEvent(owningDeploymentID string, k8sPod *v1.Pod, action central.ResourceAction) *central.SensorEvent {
	// Our current search mechanism does not support namespaced IDs, so if this is a top-level pod,
	// then having the PodID and DeploymentID fields equal will cause errors.
	// It is best to prevent this case by transforming all PodIDs.
	uid := uuid.NewV5(podNamespace, string(k8sPod.GetUID())).String()
	if action == central.ResourceAction_REMOVE_RESOURCE {
		// If we couldn't find an owning deployment ID, that means the deployment was probably removed,
		// which means the pod would have been removed from the PodStore when the owning deployment was.
		if owningDeploymentID != "" {
			d.podStore.removePod(k8sPod.GetNamespace(), owningDeploymentID, uid)
		}
		// Only the ID field is necessary for remove events.
		event := &central.SensorEvent{
			Id:     uid,
			Action: action,
			Resource: &central.SensorEvent_Pod{
				Pod: &storage.Pod{
					Id:           uid,
					Name:         k8sPod.GetName(),
					DeploymentId: owningDeploymentID,
					Namespace:    k8sPod.GetNamespace(),
				},
			},
		}
		return event
	}

	started, err := types.TimestampProto(k8sPod.GetCreationTimestamp().Time)
	if err != nil {
		log.Errorf("converting start time from Kubernetes (%v) to proto: %v", k8sPod.GetCreationTimestamp().Time, err)
	}

	p := &storage.Pod{
		Id:           uid,
		Name:         k8sPod.GetName(),
		DeploymentId: owningDeploymentID,
		ClusterId:    d.clusterID,
		Namespace:    k8sPod.Namespace,
		Started:      started,
	}

	// Assume we only receive one status per live container, so we can blindly append.
	p.LiveInstances = containerInstances(k8sPod)
	// Create a stable ordering
	sort.SliceStable(p.LiveInstances, func(i, j int) bool {
		return p.LiveInstances[i].InstanceId.Id < p.LiveInstances[j].InstanceId.Id
	})

	d.podStore.addOrUpdatePod(p)
	d.processFilter.UpdateByGivenContainers(p.DeploymentId, d.podStore.getContainersForDeployment(p.Namespace, p.DeploymentId))

	event := &central.SensorEvent{
		Id:     p.GetId(),
		Action: action,
		Resource: &central.SensorEvent_Pod{
			Pod: p,
		},
	}
	return event

}
