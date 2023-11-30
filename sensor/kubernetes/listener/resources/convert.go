package resources

import (
	"fmt"
	"reflect"
	"sort"

	"github.com/mitchellh/hashstructure/v2"
	openshiftAppsV1 "github.com/openshift/api/apps/v1"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/containers"
	imageUtils "github.com/stackrox/rox/pkg/images/utils"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/protoconv/k8s"
	"github.com/stackrox/rox/pkg/protoconv/resources"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stackrox/rox/sensor/common/service"
	"github.com/stackrox/rox/sensor/kubernetes/listener/resources/references"
	"github.com/stackrox/rox/sensor/kubernetes/orchestratornamespaces"
	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/api/batch/v1beta1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	v1listers "k8s.io/client-go/listers/core/v1"
)

const (
	k8sStandalonePodType = "StaticPods"
	kubeSystemNamespace  = "kube-system"
)

var (
	log = logging.LoggerForModule()

	k8sComponentLabelKeys = []string{
		"component",
		"k8s-app",
	}
)

func getK8sComponentID(clusterID string, component string) string {
	u, err := uuid.FromString(clusterID)
	if err != nil {
		log.Error(err)
		// ClusterID is sometimes not a valid UUID when we're doing testing,
		// so let's be forgiving in that case.
		// Unfortunately, we can't replace the entire implementation of the function with this
		// line due to backward compatibility implications.
		return uuid.NewV5FromNonUUIDs(clusterID, component).String()
	}
	return uuid.NewV5(u, component).String()
}

type deploymentWrap struct {
	*storage.Deployment
	registryOverride string
	original         interface{}
	portConfigs      map[service.PortRef]*storage.PortConfig
	pods             []*v1.Pod
	// TODO(ROX-9984): we could have the networkPoliciesApplied stored here. This would require changes in the ProcessDeployment functions of the detector.
	// networkPoliciesApplied augmentedobjs.NetworkPoliciesApplied

	// isBuilt is a flag that is used to avoid processing a deployment wrap that is stored in the in-memory store
	// while there is a `Deployment` message in the pipeline. This can happen if we received a deployment event, and
	// this event stores a new `storage.Deployment` object in in-memory store and enqueues the deployment to the resolver.
	// If, in the meantime, this deployment is read from the store (e.g. due to a related event or central events) there's
	// no way to know if the deployment is fully built. This flag is here to avoid processing a deployment that is going
	// to be processed soon (since we can guarantee that there is a deployment event in the resolver queue).
	isBuilt bool

	mutex sync.RWMutex
}

// This checks if a reflect value is a Zero value, which means the field did not exist
func doesFieldExist(value reflect.Value) bool {
	return value.IsValid()
}

func newDeploymentEventFromResource(obj interface{}, action *central.ResourceAction, deploymentType, clusterID string,
	lister v1listers.PodLister, namespaceStore *namespaceStore, hierarchy references.ParentHierarchy, registryOverride string,
	namespaces *orchestratornamespaces.OrchestratorNamespaces) *deploymentWrap {
	wrap := newWrap(obj, deploymentType, clusterID, registryOverride)
	if wrap == nil {
		return nil
	}
	if ok, err := wrap.populateNonStaticFields(obj, action, lister, namespaceStore, hierarchy, namespaces); err != nil {
		// Panic on dev because we should always be able to parse the deployments
		utils.Should(err)
		return nil
	} else if !ok {
		return nil
	}
	return wrap
}

func newWrap(obj interface{}, kind, clusterID, registryOverride string) *deploymentWrap {
	deployment, err := resources.NewDeploymentFromStaticResource(obj, kind, clusterID, registryOverride)
	if err != nil || deployment == nil {
		return nil
	}
	return &deploymentWrap{
		Deployment:       deployment,
		registryOverride: registryOverride,
		isBuilt:          false,
	}
}

func (w *deploymentWrap) populateK8sComponentIfNecessary(o *v1.Pod, hierarchy references.ParentHierarchy) *metav1.LabelSelector {
	if o.Namespace == kubeSystemNamespace {
		for _, labelKey := range k8sComponentLabelKeys {
			value, ok := o.Labels[labelKey]
			if !ok {
				continue
			}
			w.Id = getK8sComponentID(w.GetClusterId(), value)
			w.Name = fmt.Sprintf("static-%s-pods", value)
			w.Type = k8sStandalonePodType
			hierarchy.AddManually(string(o.UID), w.Id)
			return &metav1.LabelSelector{
				MatchLabels: map[string]string{
					labelKey: value,
				},
			}
		}
	}
	return nil
}

func checkIfNewPodSpecRequired(podSpec *v1.PodSpec, pods []*v1.Pod) bool {
	containerSet := set.NewStringSet()
	for _, c := range podSpec.Containers {
		containerSet.Add(c.Name)
	}
	var updated bool
	for _, p := range pods {
		if p.GetDeletionTimestamp() != nil {
			continue
		}
		for _, c := range p.Spec.Containers {
			if containerSet.Contains(c.Name) {
				continue
			}
			updated = true
			containerSet.Add(c.Name)
			podSpec.Containers = append(podSpec.Containers, c)
		}
	}
	return updated
}

func (w *deploymentWrap) populateNonStaticFields(obj interface{}, action *central.ResourceAction, lister v1listers.PodLister,
	namespaceStore *namespaceStore, hierarchy references.ParentHierarchy,
	namespaces *orchestratornamespaces.OrchestratorNamespaces) (bool, error) {
	w.original = obj
	objValue := reflect.Indirect(reflect.ValueOf(obj))
	spec := objValue.FieldByName("Spec")
	if !doesFieldExist(spec) {
		return false, fmt.Errorf("obj %+v does not have a Spec field", objValue)
	}

	var (
		podSpec       v1.PodSpec
		podLabels     map[string]string
		labelSelector *metav1.LabelSelector
		err           error
	)

	switch o := obj.(type) {
	case *openshiftAppsV1.DeploymentConfig:
		if o.Spec.Template == nil {
			return false, fmt.Errorf("spec obj %+v does not have a Template field or is not a pointer pod spec", spec)
		}
		podLabels = o.Spec.Template.Labels
		podSpec = o.Spec.Template.Spec

		labelSelector, err = w.getLabelSelector(spec)
		if err != nil {
			return false, errors.Wrap(err, "error getting label selector")
		}

	// Pods don't have the abstractions that higher level objects have, so we maintain their lifecycle independently
	case *v1.Pod:
		if o.Status.Phase == v1.PodSucceeded || o.Status.Phase == v1.PodFailed {
			*action = central.ResourceAction_REMOVE_RESOURCE
		}

		// Standalone Pods do not have a PodTemplate, like the other deployment
		// types do. So, we need to directly access the Pod's Spec field,
		// instead of looking for it inside a PodTemplate.
		podLabels = o.Labels
		labelSelector = w.populateK8sComponentIfNecessary(o, hierarchy)
	case *v1beta1.CronJob:
		// Cron jobs have a Job spec that then have a Pod Template underneath
		podLabels = o.Spec.JobTemplate.Spec.Template.GetLabels()
		podSpec = o.Spec.JobTemplate.Spec.Template.Spec
		labelSelector = o.Spec.JobTemplate.Spec.Selector
	case *batchv1.CronJob:
		// Cron jobs have a Job spec that then have a Pod Template underneath
		podLabels = o.Spec.JobTemplate.Spec.Template.GetLabels()
		podSpec = o.Spec.JobTemplate.Spec.Template.Spec
		labelSelector = o.Spec.JobTemplate.Spec.Selector
	default:
		podTemplate, err := resources.SpecToPodTemplateSpec(spec)
		if err != nil {
			return false, errors.Wrapf(err, "spec obj %+v cannot be converted to a pod template spec", spec)
		}
		podLabels = podTemplate.Labels
		podSpec = podTemplate.Spec

		labelSelector, err = w.getLabelSelector(spec)
		if err != nil {
			return false, errors.Wrap(err, "error getting label selector")
		}
	}

	labelSel, err := k8s.ToRoxLabelSelector(labelSelector)
	if err != nil {
		log.Warnf("Could not convert label selector: %v", err)
	}

	w.PodLabels = podLabels
	w.LabelSelector = labelSel
	w.AutomountServiceAccountToken = true
	if podSpec.AutomountServiceAccountToken != nil {
		w.AutomountServiceAccountToken = *podSpec.AutomountServiceAccountToken
	}

	w.populateNamespaceID(namespaceStore, namespaces)

	if labelSelector == nil {
		labelSelector = &metav1.LabelSelector{
			MatchLabels: podLabels,
		}
	}

	if *action != central.ResourceAction_REMOVE_RESOURCE {
		// If we have a standalone pod, we cannot use the labels to try and select that pod so we must directly populate the pod data
		// We need to special case kube-proxy because we are consolidating it into a deployment
		if pod, ok := obj.(*v1.Pod); ok && w.Type != k8sStandalonePodType {
			w.populateDataFromPods(set.StringSet{}, pod)
		} else {
			pods, err := w.getPods(hierarchy, labelSelector, lister)
			if err != nil {
				return false, err
			}
			if updated := checkIfNewPodSpecRequired(&podSpec, pods); updated {
				resources.NewDeploymentWrap(w.Deployment, w.registryOverride).PopulateDeploymentFromPodSpec(podSpec)
			}
			w.populateDataFromPods(set.StringSet{}, pods...)
		}
	}

	w.populatePorts()

	return true, nil
}

func (w *deploymentWrap) GetDeployment() *storage.Deployment {
	if w == nil {
		return nil
	}
	return w.Deployment
}

// Do cheap filtering on pod name based on name of higher level object (deployment, daemonset, etc)
func filterOnOwners(hierarchy references.ParentHierarchy, topLevelUID string, pods []*v1.Pod) []*v1.Pod {
	filteredPods := pods[:0]
	for _, p := range pods {
		if hierarchy.IsValidChild(topLevelUID, p) {
			filteredPods = append(filteredPods, p)
		}
	}
	return filteredPods
}

func (w *deploymentWrap) getPods(hierarchy references.ParentHierarchy, labelSelector *metav1.LabelSelector, lister v1listers.PodLister) ([]*v1.Pod, error) {
	compiledLabelSelector, err := metav1.LabelSelectorAsSelector(labelSelector)
	if err != nil {
		return nil, errors.Wrap(err, "could not compile label selector")
	}
	pods, err := lister.Pods(w.Namespace).List(compiledLabelSelector)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list pods")
	}
	return filterOnOwners(hierarchy, w.Id, pods), nil
}

func (w *deploymentWrap) populateDataFromPods(localImages set.StringSet, pods ...*v1.Pod) {
	w.pods = pods
	// Sensor must already know about the OpenShift internal registries to determine if an image is cluster-local,
	// which is ok because Sensor listens for Secrets before it starts listening for Deployment-like resources.
	w.populateImageMetadata(localImages, pods...)
}

// populateImageMetadata populates metadata for each image in the deployment.
// This metadata includes: ImageID, NotPullable, and IsClusterLocal.
// Note: NotPullable and IsClusterLocal are only determined if the image's ID can be determined.
func (w *deploymentWrap) populateImageMetadata(localImages set.StringSet, pods ...*v1.Pod) {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	// All containers have a container status
	// The downside to this is that if different pods have different versions then we will miss that fact that pods are running
	// different versions and clobber it. I've added a log to illustrate the clobbering so we can see how often it happens

	// Sort the w.Deployment.Containers by name and p.Status.ContainerStatuses by name
	// This is because the order is not guaranteed
	sort.SliceStable(w.Deployment.Containers, func(i, j int) bool {
		return w.Deployment.GetContainers()[i].Name < w.Deployment.GetContainers()[j].Name
	})

	// Sort the pods by time created as that pod will be most likely to have the most updated spec
	sort.SliceStable(pods, func(i, j int) bool {
		return pods[j].CreationTimestamp.Before(&pods[i].CreationTimestamp)
	})

	// Determine each image's ID, if not already populated, as well as if the image is pullable and/or cluster-local.
	for _, p := range pods {
		//sort.SliceStable(p.Status.ContainerStatuses, func(i, j int) bool {
		//	return p.Status.ContainerStatuses[i].Name < p.Status.ContainerStatuses[j].Name
		//})
		//sort.SliceStable(p.Spec.Containers, func(i, j int) bool {
		//	return p.Spec.Containers[i].Name < p.Spec.Containers[j].Name
		//})
		for i, c := range p.Status.ContainerStatuses {
			if i >= len(w.Deployment.Containers) || i >= len(p.Spec.Containers) {
				// This should not happen, but could happen if w.Deployment.Containers and container status are out of sync
				break
			}

			image := w.Deployment.Containers[i].Image

			// If there already is an image ID for the image then that implies that the name of the image was fully qualified
			// with an image digest. e.g. stackrox.io/main@sha256:xyz
			// If the ID already exists, populate NotPullable and IsClusterLocal.
			if image.GetId() != "" {
				// Use the image ID from the pod's ContainerStatus.
				image.NotPullable = !imageUtils.IsPullable(c.ImageID)
				image.IsClusterLocal = localImages.Contains(image.GetName().GetFullName())
				continue
			}

			parsedName, err := imageUtils.GenerateImageFromStringWithOverride(p.Spec.Containers[i].Image, w.registryOverride)
			if err != nil {
				// This error will only happen if we could not parse the image, this is possible if the image in kubernetes is malformed
				// e.g. us.gcr.io/$PROJECT/xyz:latest is an example that we have seen
				continue
			}

			// If the pod spec image doesn't match the top level image, then it is an old spec, so we should ignore its digest
			if parsedName.GetName().GetFullName() != image.GetName().GetFullName() {
				continue
			}

			if digest := imageUtils.ExtractImageDigest(c.ImageID); digest != "" {
				image.Id = digest
				image.NotPullable = !imageUtils.IsPullable(c.ImageID)
				image.IsClusterLocal = localImages.Contains(image.GetName().GetFullName())
			}
		}
	}
}

func (w *deploymentWrap) getLabelSelector(spec reflect.Value) (*metav1.LabelSelector, error) {
	s := spec.FieldByName("Selector")
	if !doesFieldExist(s) {
		return nil, nil
	}

	// Selector is of map type for replication controller
	if labelMap, ok := s.Interface().(map[string]string); ok {
		return &metav1.LabelSelector{
			MatchLabels: labelMap,
		}, nil
	}

	// All other resources uses labelSelector.
	if ls, ok := s.Interface().(*metav1.LabelSelector); ok {
		return ls, nil
	}

	return nil, fmt.Errorf("unable to get label selector for %+v", spec.Type())
}

func (w *deploymentWrap) populateNamespaceID(namespaceStore *namespaceStore,
	namespaces *orchestratornamespaces.OrchestratorNamespaces) {
	if namespaceID, found := namespaceStore.lookupNamespaceID(w.GetNamespace()); found {
		w.NamespaceId = namespaceID
		w.OrchestratorComponent = namespaces.IsOrchestratorNamespace(w.GetNamespace())
	} else {
		log.Errorf("no namespace ID found for namespace %s and deployment %q", w.GetNamespace(), w.GetName())
	}
}

func (w *deploymentWrap) populatePorts() {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	w.portConfigs = make(map[service.PortRef]*storage.PortConfig)
	for _, c := range w.GetContainers() {
		for _, p := range c.GetPorts() {
			w.portConfigs[service.PortRef{Port: intstr.FromInt(int(p.ContainerPort)), Protocol: v1.Protocol(p.Protocol)}] = p
			if p.Name != "" {
				w.portConfigs[service.PortRef{Port: intstr.FromString(p.Name), Protocol: v1.Protocol(p.Protocol)}] = p
			}
		}
	}
}

func (w *deploymentWrap) toEvent(action central.ResourceAction) *central.SensorEvent {
	w.mutex.RLock()
	defer w.mutex.RUnlock()

	return &central.SensorEvent{
		Id:     w.GetId(),
		Action: action,
		Resource: &central.SensorEvent_Deployment{
			Deployment: w.Deployment.Clone(),
		},
	}
}

// anyNonHostPort is derived from `filterHostExposure(...)`. Therefore, if `filterHostExposure(...)` is updated,
// ensure to update this function.
func (w *deploymentWrap) anyNonHostPort() bool {
	w.mutex.RLock()
	defer w.mutex.RUnlock()

	for _, portCfg := range w.portConfigs {
		for _, exposureInfo := range portCfg.ExposureInfos {
			if exposureInfo.GetLevel() != storage.PortConfig_HOST {
				return true
			}
		}
	}
	return false
}

// Make sure to update `anyNonHostPort()` if this function is updated.
func filterHostExposure(exposureInfos []*storage.PortConfig_ExposureInfo) (
	filtered []*storage.PortConfig_ExposureInfo, level storage.PortConfig_ExposureLevel,
) {
	for _, exposureInfo := range exposureInfos {
		if exposureInfo.GetLevel() != storage.PortConfig_HOST {
			continue
		}
		filtered = append(filtered, exposureInfo)
		level = storage.PortConfig_HOST
	}
	return
}

func (w *deploymentWrap) resetPortExposureNoLock() {
	for _, portCfg := range w.portConfigs {
		portCfg.ExposureInfos, portCfg.Exposure = filterHostExposure(portCfg.ExposureInfos)
	}
}

func (w *deploymentWrap) updatePortExposure(portExposure map[service.PortRef][]*storage.PortConfig_ExposureInfo) {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	w.updatePortExposureUncheckedNoLock(portExposure)
}

func (w *deploymentWrap) updatePortExposureSlice(portExposures []map[service.PortRef][]*storage.PortConfig_ExposureInfo) bool {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	w.resetPortExposureNoLock()

	changed := false
	for _, exposureInfo := range portExposures {
		if w.updatePortExposureUncheckedNoLock(exposureInfo) {
			changed = true
		}
	}
	return changed
}

func (w *deploymentWrap) updatePortExposureUncheckedNoLock(portExposure map[service.PortRef][]*storage.PortConfig_ExposureInfo) bool {
	for ref, exposureInfos := range portExposure {
		portCfg := w.portConfigs[ref]
		if portCfg == nil {
			if ref.Port.Type == intstr.String {
				// named ports MUST be defined in the pod spec
				continue
			}
			portCfg = &storage.PortConfig{
				ContainerPort: ref.Port.IntVal,
				Protocol:      string(ref.Protocol),
			}
			w.Ports = append(w.Ports, portCfg)
			w.portConfigs[ref] = portCfg
		}

		portCfg.ExposureInfos = append(portCfg.ExposureInfos, exposureInfos...)

		for _, exposureInfo := range exposureInfos {
			if containers.CompareExposureLevel(portCfg.Exposure, exposureInfo.GetLevel()) < 0 {
				portCfg.Exposure = exposureInfo.GetLevel()
			}
		}
	}
	for _, portCfg := range w.portConfigs {
		sort.Slice(portCfg.ExposureInfos, func(i, j int) bool {
			return portCfg.ExposureInfos[i].ServiceName < portCfg.ExposureInfos[j].ServiceName
		})
	}

	sort.Slice(w.Ports, func(i, j int) bool {
		return w.Ports[i].ContainerPort < w.Ports[j].ContainerPort
	})
	return false
}

func (w *deploymentWrap) updateServiceAccountPermissionLevel(permissionLevel storage.PermissionLevel) bool {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	if w.ServiceAccountPermissionLevel != permissionLevel {
		w.ServiceAccountPermissionLevel = permissionLevel
		return true
	}
	return false
}

// Clone clones a deploymentWrap. Note: `original` field is not cloned.
func (w *deploymentWrap) Clone() *deploymentWrap {
	w.mutex.RLock()
	defer w.mutex.RUnlock()

	ret := &deploymentWrap{
		original:         w.original, // original is only always read
		registryOverride: w.registryOverride,
		Deployment:       w.GetDeployment().Clone(),
	}
	if w.pods != nil {
		ret.pods = make([]*v1.Pod, len(w.pods))
		for idx, pod := range w.pods {
			ret.pods[idx] = pod.DeepCopy()
		}
	}
	if w.portConfigs != nil {
		ret.portConfigs = make(map[service.PortRef]*storage.PortConfig)
		for k, v := range w.portConfigs {
			ret.portConfigs[k] = v.Clone()
		}
	}

	return ret
}

func (w *deploymentWrap) updateHash() error {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	hashValue, err := hashstructure.Hash(w.GetDeployment(), hashstructure.FormatV2, &hashstructure.HashOptions{})
	if err != nil {
		return errors.Wrap(err, "calculating deployment hash")
	}
	w.Hash = hashValue
	return nil
}
