package resources

import (
	"fmt"
	"reflect"
	"sort"
	"strings"

	openshift_appsv1 "github.com/openshift/api/apps/v1"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/containers"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/images/types"
	imageUtils "github.com/stackrox/rox/pkg/images/utils"
	"github.com/stackrox/rox/pkg/kubernetes"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/protoconv/k8s"
	"github.com/stackrox/rox/pkg/protoconv/resources"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/uuid"
	"k8s.io/api/batch/v1beta1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
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

func getK8sComponentID(component string) string {
	u, err := uuid.FromString(env.ClusterID.Setting())
	if err != nil {
		log.Error(err)
		return ""
	}
	return uuid.NewV5(u, component).String()
}

type deploymentWrap struct {
	*storage.Deployment
	original    interface{}
	portConfigs map[portRef]*storage.PortConfig
	pods        []*v1.Pod
	podSelector labels.Selector
}

// This checks if a reflect value is a Zero value, which means the field did not exist
func doesFieldExist(value reflect.Value) bool {
	return !reflect.DeepEqual(value, reflect.Value{})
}

func newDeploymentEventFromResource(obj interface{}, action central.ResourceAction, deploymentType string, lister v1listers.PodLister, namespaceStore *namespaceStore) *deploymentWrap {
	wrap := newWrap(obj, deploymentType)
	if wrap == nil {
		return nil
	}
	if err := wrap.populateNonStaticFields(obj, action, lister, namespaceStore); err != nil {
		log.Error(err)
		return nil
	}
	return wrap
}

func newWrap(obj interface{}, kind string) *deploymentWrap {
	deployment, err := resources.NewDeploymentFromStaticResource(obj, kind)
	if err != nil || deployment == nil {
		return nil
	}
	return &deploymentWrap{
		Deployment: deployment,
	}
}

func (w *deploymentWrap) populateK8sComponentIfNecessary(o *v1.Pod) *metav1.LabelSelector {
	if o.Namespace == kubeSystemNamespace {
		for _, labelKey := range k8sComponentLabelKeys {
			value, ok := o.Labels[labelKey]
			if !ok {
				continue
			}
			w.Id = getK8sComponentID(value)
			w.Name = fmt.Sprintf("static-%s-pods", value)
			w.Type = k8sStandalonePodType
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

func (w *deploymentWrap) populateNonStaticFields(obj interface{}, action central.ResourceAction, lister v1listers.PodLister, namespaceStore *namespaceStore) error {
	w.original = obj
	objValue := reflect.Indirect(reflect.ValueOf(obj))
	spec := objValue.FieldByName("Spec")
	if !doesFieldExist(spec) {
		return fmt.Errorf("obj %+v does not have a Spec field", objValue)
	}

	var (
		podSpec       v1.PodSpec
		podLabels     map[string]string
		labelSelector *metav1.LabelSelector
		err           error
	)

	switch o := obj.(type) {
	case *openshift_appsv1.DeploymentConfig:
		if o.Spec.Template == nil {
			return fmt.Errorf("spec obj %+v does not have a Template field or is not a pointer pod spec", spec)
		}
		podLabels = o.Spec.Template.Labels
		podSpec = o.Spec.Template.Spec

		labelSelector, err = w.getLabelSelector(spec)
		if err != nil {
			return errors.Wrap(err, "error getting label selector")
		}

	// Pods don't have the abstractions that higher level objects have so maintain it's lifecycle independently
	case *v1.Pod:
		if o.Status.Phase != v1.PodRunning {
			return fmt.Errorf("found Pod %s, but it was not running", o.Name)
		}
		// Standalone Pods do not have a PodTemplate, like the other deployment
		// types do. So, we need to directly access the Pod's Spec field,
		// instead of looking for it inside a PodTemplate.
		podLabels = o.Labels
		labelSelector = w.populateK8sComponentIfNecessary(o)

	case *v1beta1.CronJob:
		// Cron jobs have a Job spec that then have a Pod Template underneath
		podLabels = o.Spec.JobTemplate.Spec.Template.GetLabels()
		podSpec = o.Spec.JobTemplate.Spec.Template.Spec
		labelSelector = o.Spec.JobTemplate.Spec.Selector

	default:
		podTemplate, ok := spec.FieldByName("Template").Interface().(v1.PodTemplateSpec)
		if !ok {
			return fmt.Errorf("spec obj %+v does not have a Template field", spec)
		}
		podLabels = podTemplate.Labels
		podSpec = podTemplate.Spec

		labelSelector, err = w.getLabelSelector(spec)
		if err != nil {
			return errors.Wrap(err, "error getting label selector")
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

	w.populateNamespaceID(namespaceStore)

	if labelSelector == nil {
		labelSelector = &metav1.LabelSelector{
			MatchLabels: podLabels,
		}
	}

	if action != central.ResourceAction_REMOVE_RESOURCE {
		// If we have a standalone pod, we cannot use the labels to try and select that pod so we must directly populate the pod data
		// We need to special case kube-proxy because we are consolidating it into a deployment
		if pod, ok := obj.(*v1.Pod); ok && w.Type != k8sStandalonePodType {
			w.populatePorts()
			w.populateDataFromPods(pod)
		} else {
			pods, err := w.getPods(w.Name, labelSelector, lister)
			if err != nil {
				return err
			}
			if updated := checkIfNewPodSpecRequired(&podSpec, pods); updated {
				resources.NewDeploymentWrap(w.Deployment).PopulateDeploymentFromPodSpec(podSpec)
			}
			w.populatePorts()
			w.populateDataFromPods(pods...)
		}
	} else {
		w.populatePorts()
	}
	return nil
}

func (w *deploymentWrap) GetDeployment() *storage.Deployment {
	if w == nil {
		return nil
	}
	return w.Deployment
}

func matchesOwnerName(name string, p *v1.Pod) bool {
	// Edge case that happens for Standalone Pods
	if len(p.GetOwnerReferences()) == 0 {
		return true
	}
	kind := p.GetOwnerReferences()[0].Kind
	var numExpectedDashes int
	switch kind {
	case kubernetes.ReplicaSet, kubernetes.CronJob, kubernetes.Job, kubernetes.Deployment, kubernetes.DeploymentConfig: // 2 dash in pod
		// nginx-deployment-86d59dd769-7gmsk we want nginx-deployment
		numExpectedDashes = 2
	case kubernetes.DaemonSet, kubernetes.StatefulSet, kubernetes.ReplicationController: // 1 dash in pod
		// nginx-deployment-7gmsk we want nginx-deployment
		numExpectedDashes = 1
	default:
		log.Warnf("Currently do not handle owner kind %q. Attributing the pod", kind)
		// By default if we can't parse, then we'll hit the mis-attribution edge case, but I'd rather do that
		// then miss the pods altogether
		return true
	}
	if spl := strings.Split(p.GetName(), "-"); len(spl) > numExpectedDashes {
		return name == strings.Join(spl[:len(spl)-numExpectedDashes], "-")
	}
	log.Warnf("Could not parse pod %q with owner type %q", p.GetName(), kind)
	return false
}

// Do cheap filtering on pod name based on name of higher level object (deployment, daemonset, etc)
func filterOnName(name string, pods []*v1.Pod) []*v1.Pod {
	filteredPods := pods[:0]
	for _, p := range pods {
		if matchesOwnerName(name, p) {
			filteredPods = append(filteredPods, p)
		}
	}
	return filteredPods
}

func (w *deploymentWrap) getPods(topLevelName string, labelSelector *metav1.LabelSelector, lister v1listers.PodLister) ([]*v1.Pod, error) {
	compiledLabelSelector, err := metav1.LabelSelectorAsSelector(labelSelector)
	if err != nil {
		return nil, errors.Wrap(err, "could not compile label selector")
	}
	w.podSelector = compiledLabelSelector
	pods, err := lister.Pods(w.Namespace).List(w.podSelector)
	if err != nil {
		return nil, err
	}
	return filterOnName(topLevelName, pods), nil
}

func (w *deploymentWrap) populateDataFromPods(pods ...*v1.Pod) {
	w.pods = pods
	w.populateImageShas(pods...)
	w.populateContainerInstances(pods...)
}

func (w *deploymentWrap) populateContainerInstances(pods ...*v1.Pod) {
	for _, p := range pods {
		for i, instance := range containerInstances(p) {
			// This check that the size is not greater is necessary, because pods can be in terminating as a deployment is updated
			// The deployment will still be managing the pods, but we want to take the new pod(s) as the source of truth
			if i >= len(w.Containers) {
				break
			}
			w.Containers[i].Instances = append(w.Containers[i].Instances, instance)
		}
	}
}

func (w *deploymentWrap) populateImageShas(pods ...*v1.Pod) {
	// All containers have a container status
	// The downside to this is that if different pods have different versions then we will miss that fact that pods are running
	// different versions and clobber it. I've added a log to illustrate the clobbering so we can see how often it happens

	// Sort the w.Deployment.Containers by name and p.Status.ContainerStatuses by name
	// This is because the order is not guaranteed
	sort.SliceStable(w.Deployment.Containers, func(i, j int) bool {
		return w.Deployment.GetContainers()[i].Name < w.Deployment.GetContainers()[j].Name
	})
	for _, p := range pods {
		sort.SliceStable(p.Status.ContainerStatuses, func(i, j int) bool {
			return p.Status.ContainerStatuses[i].Name < p.Status.ContainerStatuses[j].Name
		})
		for i, c := range p.Status.ContainerStatuses {
			if i >= len(w.Deployment.Containers) {
				// This should not happened, but could happen if w.Deployment.Containers and container status are out of sync
				break
			}
			if sha := imageUtils.ExtractImageSha(c.ImageID); sha != "" {
				sha = types.NewDigest(sha).Digest()
				w.Deployment.Containers[i].Image.Id = types.NewDigest(sha).Digest()
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

func (w *deploymentWrap) populateNamespaceID(namespaceStore *namespaceStore) {
	if namespaceID, found := namespaceStore.lookupNamespaceID(w.GetNamespace()); found {
		w.NamespaceId = namespaceID
	}
}

func (w *deploymentWrap) populatePorts() {
	w.portConfigs = make(map[portRef]*storage.PortConfig)
	for _, c := range w.GetContainers() {
		for _, p := range c.GetPorts() {
			w.portConfigs[portRef{Port: intstr.FromInt(int(p.ContainerPort)), Protocol: v1.Protocol(p.Protocol)}] = p
			if p.Name != "" {
				w.portConfigs[portRef{Port: intstr.FromString(p.Name), Protocol: v1.Protocol(p.Protocol)}] = p
			}
		}
	}
}

func (w *deploymentWrap) toEvent(action central.ResourceAction) *central.SensorEvent {
	return &central.SensorEvent{
		Id:     w.GetId(),
		Action: action,
		Resource: &central.SensorEvent_Deployment{
			Deployment: w.Deployment,
		},
	}
}

func filterHostExposure(exposureInfos []*storage.PortConfig_ExposureInfo) (filtered []*storage.PortConfig_ExposureInfo, level storage.PortConfig_ExposureLevel) {
	for _, exposureInfo := range exposureInfos {
		if exposureInfo.GetLevel() != storage.PortConfig_HOST {
			continue
		}
		filtered = append(filtered, exposureInfo)
		level = storage.PortConfig_HOST
	}
	return
}

func (w *deploymentWrap) resetPortExposure() {
	for _, portCfg := range w.portConfigs {
		portCfg.ExposureInfos, portCfg.Exposure = filterHostExposure(portCfg.ExposureInfos)
	}
}

func (w *deploymentWrap) updatePortExposureFromStore(store *serviceStore) {
	w.resetPortExposure()

	svcs := store.getMatchingServices(w.Namespace, w.PodLabels)
	for _, svc := range svcs {
		w.updatePortExposure(svc)
	}
}

func (w *deploymentWrap) updatePortExposure(svc *serviceWrap) {
	if !svc.selector.Matches(labels.Set(w.PodLabels)) {
		return
	}

	for ref, exposureInfo := range svc.exposure() {
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

		portCfg.ExposureInfos = append(portCfg.ExposureInfos, exposureInfo)

		if containers.CompareExposureLevel(portCfg.Exposure, exposureInfo.GetLevel()) < 0 {
			portCfg.Exposure = exposureInfo.GetLevel()
		}
	}
}
