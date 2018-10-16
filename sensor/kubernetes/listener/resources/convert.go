package resources

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strconv"

	ptypes "github.com/gogo/protobuf/types"
	openshift_appsv1 "github.com/openshift/api/apps/v1"
	pkgV1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/containers"
	"github.com/stackrox/rox/pkg/images/types"
	imageUtils "github.com/stackrox/rox/pkg/images/utils"
	"github.com/stackrox/rox/pkg/kubernetes"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/sensor/kubernetes/volumes"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/intstr"
	v1listers "k8s.io/client-go/listers/core/v1"
)

const (
	openshiftEncodedDeploymentConfigAnnotation = `openshift.io/encoded-deployment-config`

	megabyte = 1024 * 1024
)

var logger = logging.LoggerForModule()

type deploymentWrap struct {
	*pkgV1.Deployment
	original    interface{}
	podLabels   map[string]string
	portConfigs map[portRef]*pkgV1.PortConfig
	pods        []*v1.Pod
	podSelector labels.Selector
}

// This checks if a reflect value is a Zero value, which means the field did not exist
func doesFieldExist(value reflect.Value) bool {
	return !reflect.DeepEqual(value, reflect.Value{})
}

func newDeploymentEventFromResource(obj interface{}, action pkgV1.ResourceAction, deploymentType string, lister v1listers.PodLister) (wrap *deploymentWrap) {
	objMeta, err := meta.Accessor(obj)
	if err != nil {
		logger.Errorf("could not access metadata of object of type %T: %v", obj, err)
		return
	}
	kind := deploymentType

	// Ignore resources that are owned by another resource.
	// DeploymentConfigs can be owned by TemplateInstance which we don't care about
	if len(objMeta.GetOwnerReferences()) > 0 && kind != kubernetes.DeploymentConfig {
		return
	}

	// This only applies to OpenShift
	if encDeploymentConfig, ok := objMeta.GetLabels()[openshiftEncodedDeploymentConfigAnnotation]; ok {
		newMeta, newKind, err := extractDeploymentConfig(encDeploymentConfig)
		if err != nil {
			logger.Error(err)
		} else {
			objMeta, kind = newMeta, newKind
		}
	}

	wrap = newWrap(objMeta, kind)
	wrap.populateFields(obj, action, lister)
	return
}

func extractDeploymentConfig(encodedDeploymentConfig string) (metav1.Object, string, error) {
	// Anonymous struct that only contains the fields we are interested in (note: json.Unmarshal silently ignores
	// fields that are not in the destination object).
	dc := struct {
		metav1.TypeMeta
		MetaData metav1.ObjectMeta `json:"metadata"`
	}{}
	err := json.Unmarshal([]byte(encodedDeploymentConfig), &dc)
	return &dc.MetaData, dc.Kind, err
}

func newWrap(meta metav1.Object, kind string) *deploymentWrap {
	updatedTime, err := ptypes.TimestampProto(meta.GetCreationTimestamp().Time)
	if err != nil {
		logger.Error(err)
	}
	return &deploymentWrap{
		Deployment: &pkgV1.Deployment{
			Id:          string(meta.GetUID()),
			Name:        meta.GetName(),
			Type:        kind,
			Version:     meta.GetResourceVersion(),
			Namespace:   meta.GetNamespace(),
			Labels:      meta.GetLabels(),
			Annotations: meta.GetAnnotations(),
			UpdatedAt:   updatedTime,
		},
	}
}

func (w *deploymentWrap) populateFields(obj interface{}, action pkgV1.ResourceAction, lister v1listers.PodLister) {
	w.original = obj
	objValue := reflect.Indirect(reflect.ValueOf(obj))
	spec := objValue.FieldByName("Spec")
	if !doesFieldExist(spec) {
		logger.Errorf("Obj %+v does not have a Spec field", objValue)
		return
	}

	w.populateReplicas(spec)

	var podSpec v1.PodSpec
	var podLabels map[string]string

	switch o := obj.(type) {
	case *openshift_appsv1.DeploymentConfig:
		if o.Spec.Template == nil {
			logger.Errorf("Spec obj %+v does not have a Template field or is not a pointer pod spec", spec)
			return
		}
		podSpec = o.Spec.Template.Spec
		podLabels = o.Spec.Template.Labels
	// Pods don't have the abstractions that higher level objects have so maintain it's lifecycle independently
	case *v1.Pod:
		// Standalone Pods do not have a PodTemplate, like the other deployment
		// types do. So, we need to directly access the Pod's Spec field,
		// instead of looking for it inside a PodTemplate.
		podSpec = o.Spec
		podLabels = o.Labels
	default:
		podTemplate, ok := spec.FieldByName("Template").Interface().(v1.PodTemplateSpec)
		if !ok {
			logger.Errorf("Spec obj %+v does not have a Template field", spec)
			return
		}
		podSpec = podTemplate.Spec
		podLabels = podTemplate.Labels
	}

	w.podLabels = podLabels
	w.populateContainers(podSpec)

	if action != pkgV1.ResourceAction_REMOVE_RESOURCE {
		// If we have a standalone pod, we cannot use the labels to try and select that pod so we must directly populate the pod data
		if pod, ok := obj.(*v1.Pod); ok {
			w.populateDataFromPods(pod)
		} else {
			if err := w.populatePodData(spec, lister); err != nil {
				logger.Errorf("Could not populate pod data: %v", err)
			}
		}
	}
}

func (w *deploymentWrap) GetDeployment() *pkgV1.Deployment {
	if w == nil {
		return nil
	}
	return w.Deployment
}

func (w *deploymentWrap) populateContainers(podSpec v1.PodSpec) {
	w.Deployment.Containers = make([]*pkgV1.Container, 0, len(podSpec.Containers))
	for _, c := range podSpec.Containers {
		w.Deployment.Containers = append(w.Deployment.Containers, &pkgV1.Container{
			Name: c.Name,
		})
	}

	w.populateServiceAccount(podSpec)
	w.populateContainerConfigs(podSpec)
	w.populateImages(podSpec)
	w.populateSecurityContext(podSpec)
	w.populateVolumesAndSecrets(podSpec)
	w.populatePorts(podSpec)
	w.populateResources(podSpec)
	w.populateImagePullSecrets(podSpec)
}

func (w *deploymentWrap) populateServiceAccount(podSpec v1.PodSpec) {
	w.ServiceAccount = podSpec.ServiceAccountName
}

func (w *deploymentWrap) populateImagePullSecrets(podSpec v1.PodSpec) {
	secrets := make([]string, 0, len(podSpec.ImagePullSecrets))
	for _, s := range podSpec.ImagePullSecrets {
		secrets = append(secrets, s.Name)
	}
	w.ImagePullSecrets = secrets
}

func (w *deploymentWrap) populateReplicas(spec reflect.Value) {
	replicaField := spec.FieldByName("Replicas")
	if !doesFieldExist(replicaField) {
		return
	}

	replicasPointer, ok := replicaField.Interface().(*int32)
	if ok && replicasPointer != nil {
		w.Deployment.Replicas = int64(*replicasPointer)
	}

	replicas, ok := replicaField.Interface().(int32)
	if ok {
		w.Deployment.Replicas = int64(replicas)
	}
}

func (w *deploymentWrap) populatePodData(spec reflect.Value, lister v1listers.PodLister) error {
	labelSelector, err := w.getLabelSelector(spec)
	if err != nil {
		return err
	}
	w.podSelector = labelSelector
	w.pods, err = lister.Pods(w.Namespace).List(w.podSelector)
	if err != nil {
		return err
	}
	w.populateDataFromPods(w.pods...)
	return nil
}

func (w *deploymentWrap) populateDataFromPods(pods ...*v1.Pod) {
	w.populateImageShas(pods...)
	w.populateContainerInstances(pods...)
}

func (w *deploymentWrap) populateContainerInstances(pods ...*v1.Pod) {
	for _, p := range pods {
		for i, instance := range containerInstances(p) {
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
				// Logging to see that we are clobbering a value from an old sha
				currentSHA := w.Deployment.GetContainers()[i].GetImage().GetId()
				if currentSHA != "" && currentSHA != sha {
					logger.Warnf("Clobbering SHA '%s' found for image '%s' with SHA '%s'", currentSHA, c.Image, sha)
				}
				w.Deployment.Containers[i].Image.Id = types.NewDigest(sha).Digest()
			}
		}
	}
}

func (w *deploymentWrap) getLabelSelector(spec reflect.Value) (labels.Selector, error) {
	s := spec.FieldByName("Selector")
	if !doesFieldExist(s) {
		return labels.Nothing(), nil
	}

	// Selector is of map type for replication controller
	if labelMap, ok := s.Interface().(map[string]string); ok {
		return SelectorFromMap(labelMap), nil
	}

	// All other resources uses labelSelector.
	if ls, ok := s.Interface().(*metav1.LabelSelector); ok {
		return metav1.LabelSelectorAsSelector(ls)
	}

	return nil, fmt.Errorf("unable to get label selector for %+v", spec.Type())
}

func (w *deploymentWrap) populateContainerConfigs(podSpec v1.PodSpec) {
	for i, c := range podSpec.Containers {

		// Skip if there's nothing to add.
		if len(c.Command) == 0 && len(c.Args) == 0 && len(c.WorkingDir) == 0 && len(c.Env) == 0 && c.SecurityContext == nil {
			continue
		}

		config := &pkgV1.ContainerConfig{
			Command:   c.Command,
			Args:      c.Args,
			Directory: c.WorkingDir,
		}

		envSlice := make([]*pkgV1.ContainerConfig_EnvironmentConfig, len(c.Env))
		for i, env := range c.Env {
			envSlice[i] = &pkgV1.ContainerConfig_EnvironmentConfig{
				Key:   env.Name,
				Value: env.Value,
			}
		}

		config.Env = envSlice

		if s := c.SecurityContext; s != nil {
			if uid := s.RunAsUser; uid != nil {
				config.Uid = *uid
			}
		}

		w.Deployment.Containers[i].Id = w.Deployment.Id + ":" + c.Name
		w.Deployment.Containers[i].Config = config
	}
}

func (w *deploymentWrap) populateImages(podSpec v1.PodSpec) {
	for i, c := range podSpec.Containers {
		w.Deployment.Containers[i].Image = imageUtils.GenerateImageFromString(c.Image)
	}
}

func (w *deploymentWrap) populateSecurityContext(podSpec v1.PodSpec) {
	for i, c := range podSpec.Containers {
		if s := c.SecurityContext; s != nil {
			sc := &pkgV1.SecurityContext{}

			if p := s.Privileged; p != nil {
				sc.Privileged = *p
			}

			if SELinux := s.SELinuxOptions; SELinux != nil {
				sc.Selinux = &pkgV1.SecurityContext_SELinux{
					User:  SELinux.User,
					Role:  SELinux.Role,
					Type:  SELinux.Type,
					Level: SELinux.Level,
				}
			}

			if capabilities := s.Capabilities; capabilities != nil {
				for _, add := range capabilities.Add {
					sc.AddCapabilities = append(sc.AddCapabilities, string(add))
				}

				for _, drop := range capabilities.Drop {
					sc.DropCapabilities = append(sc.DropCapabilities, string(drop))
				}
			}

			w.Deployment.Containers[i].SecurityContext = sc
		}
	}
}

func (w *deploymentWrap) getVolumeSourceMap(podSpec v1.PodSpec) map[string]volumes.VolumeSource {
	volumeSourceMap := make(map[string]volumes.VolumeSource)
	for _, v := range podSpec.Volumes {
		val := reflect.ValueOf(v.VolumeSource)
		for i := 0; i < val.NumField(); i++ {
			f := val.Field(i)
			if !f.IsNil() {
				sourceCreator, ok := volumes.VolumeRegistry[val.Type().Field(i).Name]
				if !ok {
					volumeSourceMap[v.Name] = &volumes.Unimplemented{}
				} else {
					volumeSourceMap[v.Name] = sourceCreator(f.Interface())
				}
			}
		}
	}
	return volumeSourceMap
}

func convertQuantityToCores(q *resource.Quantity) float32 {
	// kubernetes does not like floating point values so they make you jump through hoops
	f, err := strconv.ParseFloat(q.AsDec().String(), 32)
	if err != nil {
		logger.Error(err)
	}
	return float32(f)
}

func convertQuantityToMb(q *resource.Quantity) float32 {
	return float32(float64(q.Value()) / megabyte)
}

func (w *deploymentWrap) populateResources(podSpec v1.PodSpec) {
	for i, c := range podSpec.Containers {
		w.Deployment.Containers[i].Resources = &pkgV1.Resources{
			CpuCoresRequest: convertQuantityToCores(c.Resources.Requests.Cpu()),
			CpuCoresLimit:   convertQuantityToCores(c.Resources.Limits.Cpu()),
			MemoryMbRequest: convertQuantityToMb(c.Resources.Requests.Memory()),
			MemoryMbLimit:   convertQuantityToMb(c.Resources.Limits.Memory()),
		}
	}
}

func (w *deploymentWrap) populateVolumesAndSecrets(podSpec v1.PodSpec) {
	volumeSourceMap := w.getVolumeSourceMap(podSpec)
	for i, c := range podSpec.Containers {
		for _, v := range c.VolumeMounts {
			sourceVolume, ok := volumeSourceMap[v.Name]
			if !ok {
				sourceVolume = &volumes.Unimplemented{}
			}
			if sourceVolume.Type() == "Secret" {
				w.Deployment.Containers[i].Secrets = append(w.Deployment.Containers[i].Secrets, &pkgV1.EmbeddedSecret{
					Name: sourceVolume.Source(),
					Path: v.MountPath,
				})
				continue
			}
			w.Deployment.Containers[i].Volumes = append(w.Deployment.Containers[i].Volumes, &pkgV1.Volume{
				Name:        v.Name,
				Source:      sourceVolume.Source(),
				Destination: v.MountPath,
				ReadOnly:    v.ReadOnly,
				Type:        sourceVolume.Type(),
			})
		}
	}
}

func (w *deploymentWrap) populatePorts(podSpec v1.PodSpec) {
	w.portConfigs = make(map[portRef]*pkgV1.PortConfig)
	for i, c := range podSpec.Containers {
		for _, p := range c.Ports {
			exposedPort := p.ContainerPort
			// If the port defines a host port, then it is exposed via that port instead of the container port
			if p.HostPort != 0 {
				exposedPort = p.HostPort
			}

			portConfig := &pkgV1.PortConfig{
				Name:          p.Name,
				ContainerPort: p.ContainerPort,
				ExposedPort:   exposedPort,
				Protocol:      string(p.Protocol),
				Exposure:      pkgV1.PortConfig_INTERNAL,
			}
			w.Deployment.Containers[i].Ports = append(w.Deployment.Containers[i].Ports, portConfig)
			w.portConfigs[portRef{Port: intstr.FromInt(int(p.ContainerPort)), Protocol: p.Protocol}] = portConfig
			if p.Name != "" {
				w.portConfigs[portRef{Port: intstr.FromString(p.Name), Protocol: p.Protocol}] = portConfig
			}
		}
	}
}

func (w *deploymentWrap) toEvent(action pkgV1.ResourceAction) *pkgV1.SensorEvent {
	return &pkgV1.SensorEvent{
		Id:     w.GetId(),
		Action: action,
		Resource: &pkgV1.SensorEvent_Deployment{
			Deployment: w.Deployment,
		},
	}
}

func (w *deploymentWrap) resetPortExposure() (updated bool) {
	for _, portCfg := range w.portConfigs {
		if portCfg.Exposure != pkgV1.PortConfig_INTERNAL {
			portCfg.Exposure = pkgV1.PortConfig_INTERNAL
			updated = true
		}
	}
	return
}

func (w *deploymentWrap) updatePortExposureFromStore(store *serviceStore) (updated bool) {
	updated = w.resetPortExposure()

	svcs := store.getMatchingServices(w.Namespace, w.podLabels)
	for _, svc := range svcs {
		updated = w.updatePortExposure(svc) || updated
	}
	return
}

func (w *deploymentWrap) updatePortExposure(svc *serviceWrap) (updated bool) {
	if !svc.selector.Matches(labels.Set(w.podLabels)) {
		return
	}

	exposure := svc.exposure()
	for _, svcPort := range svc.Spec.Ports {
		portCfg := w.portConfigs[portRef{Port: svcPort.TargetPort, Protocol: svcPort.Protocol}]
		if portCfg == nil {
			continue
		}
		if containers.IncreasedExposureLevel(portCfg.Exposure, exposure) {
			portCfg.Exposure = exposure
			updated = true
		}
	}
	return
}
