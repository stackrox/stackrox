package resources

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"

	ptypes "github.com/gogo/protobuf/types"
	openshift_appsv1 "github.com/openshift/api/apps/v1"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/images/types"
	imageUtils "github.com/stackrox/rox/pkg/images/utils"
	"github.com/stackrox/rox/pkg/kubernetes"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/protoconv/resources/volumes"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	openshiftEncodedDeploymentConfigAnnotation = `openshift.io/encoded-deployment-config`

	megabyte = 1024 * 1024
)

var (
	logger = logging.LoggerForModule()
)

// DeploymentWrap is a wrapper around a deployment to help convert the static fields
type DeploymentWrap struct {
	*storage.Deployment
}

// This checks if a reflect value is a Zero value, which means the field did not exist
func doesFieldExist(value reflect.Value) bool {
	return !reflect.DeepEqual(value, reflect.Value{})
}

// NewDeploymentFromStaticResource returns a storage.Deployment from a k8s object
func NewDeploymentFromStaticResource(obj interface{}, deploymentType string) (*storage.Deployment, error) {
	objMeta, err := meta.Accessor(obj)
	if err != nil {
		return nil, fmt.Errorf("could not access metadata of object of type %T: %v", obj, err)
	}
	kind := deploymentType

	// Ignore resources that are owned by another resource.
	// DeploymentConfigs can be owned by TemplateInstance which we don't care about
	if len(objMeta.GetOwnerReferences()) > 0 && kind != kubernetes.DeploymentConfig {
		return nil, nil
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

	wrap := newWrap(objMeta, kind)
	wrap.populateFields(obj)
	return wrap.Deployment, nil

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

func newWrap(meta metav1.Object, kind string) *DeploymentWrap {
	updatedTime, err := ptypes.TimestampProto(meta.GetCreationTimestamp().Time)
	if err != nil {
		logger.Error(err)
	}
	return &DeploymentWrap{
		Deployment: &storage.Deployment{
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

func (w *DeploymentWrap) populateFields(obj interface{}) {
	objValue := reflect.Indirect(reflect.ValueOf(obj))
	spec := objValue.FieldByName("Spec")
	if !doesFieldExist(spec) {
		logger.Errorf("Obj %+v does not have a Spec field", objValue)
		return
	}

	w.populateReplicas(spec)

	var podSpec v1.PodSpec

	switch o := obj.(type) {
	case *openshift_appsv1.DeploymentConfig:
		if o.Spec.Template == nil {
			logger.Errorf("Spec obj %+v does not have a Template field or is not a pointer pod spec", spec)
			return
		}
		podSpec = o.Spec.Template.Spec
		// Pods don't have the abstractions that higher level objects have so maintain it's lifecycle independently
	case *v1.Pod:
		// Standalone Pods do not have a PodTemplate, like the other deployment
		// types do. So, we need to directly access the Pod's Spec field,
		// instead of looking for it inside a PodTemplate.
		podSpec = o.Spec
	default:
		podTemplate, ok := spec.FieldByName("Template").Interface().(v1.PodTemplateSpec)
		if !ok {
			logger.Errorf("Spec obj %+v does not have a Template field", spec)
			return
		}
		podSpec = podTemplate.Spec
	}

	w.HostNetwork = podSpec.HostNetwork
	w.populateContainers(podSpec)
}

func (w *DeploymentWrap) populateContainers(podSpec v1.PodSpec) {
	w.Deployment.Containers = make([]*storage.Container, 0, len(podSpec.Containers))
	for _, c := range podSpec.Containers {
		w.Deployment.Containers = append(w.Deployment.Containers, &storage.Container{
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

func (w *DeploymentWrap) populateServiceAccount(podSpec v1.PodSpec) {
	w.ServiceAccount = podSpec.ServiceAccountName
}

func (w *DeploymentWrap) populateImagePullSecrets(podSpec v1.PodSpec) {
	secrets := make([]string, 0, len(podSpec.ImagePullSecrets))
	for _, s := range podSpec.ImagePullSecrets {
		secrets = append(secrets, s.Name)
	}
	w.ImagePullSecrets = secrets
}

func (w *DeploymentWrap) populateReplicas(spec reflect.Value) {
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
		logger.Warnf("Currently do not handle owner kind %q. Attributing the pod", kind)
		// By default if we can't parse, then we'll hit the mis-attribution edge case, but I'd rather do that
		// then miss the pods altogether
		return true
	}
	if spl := strings.Split(p.GetName(), "-"); len(spl) > numExpectedDashes {
		return name == strings.Join(spl[:len(spl)-numExpectedDashes], "-")
	}
	logger.Warnf("Could not parse pod %q with owner type %q", p.GetName(), kind)
	return false
}

func (w *DeploymentWrap) populateDataFromPods(pods ...*v1.Pod) {
	w.populateImageShas(pods...)
}

func (w *DeploymentWrap) populateImageShas(pods ...*v1.Pod) {
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

func (w *DeploymentWrap) populateContainerConfigs(podSpec v1.PodSpec) {
	for i, c := range podSpec.Containers {

		// Skip if there's nothing to add.
		if len(c.Command) == 0 && len(c.Args) == 0 && len(c.WorkingDir) == 0 && len(c.Env) == 0 && c.SecurityContext == nil {
			continue
		}

		config := &storage.ContainerConfig{
			Command:   c.Command,
			Args:      c.Args,
			Directory: c.WorkingDir,
		}

		envSlice := make([]*storage.ContainerConfig_EnvironmentConfig, len(c.Env))
		for i, env := range c.Env {
			envSlice[i] = &storage.ContainerConfig_EnvironmentConfig{
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

func (w *DeploymentWrap) populateImages(podSpec v1.PodSpec) {
	for i, c := range podSpec.Containers {
		w.Deployment.Containers[i].Image = imageUtils.GenerateImageFromString(c.Image)
	}
}

func (w *DeploymentWrap) populateSecurityContext(podSpec v1.PodSpec) {
	for i, c := range podSpec.Containers {
		if s := c.SecurityContext; s != nil {
			sc := &storage.SecurityContext{}

			if p := s.Privileged; p != nil {
				sc.Privileged = *p
			}

			if SELinux := s.SELinuxOptions; SELinux != nil {
				sc.Selinux = &storage.SecurityContext_SELinux{
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

func (w *DeploymentWrap) getVolumeSourceMap(podSpec v1.PodSpec) map[string]volumes.VolumeSource {
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

func (w *DeploymentWrap) populateResources(podSpec v1.PodSpec) {
	for i, c := range podSpec.Containers {
		w.Deployment.Containers[i].Resources = &storage.Resources{
			CpuCoresRequest: convertQuantityToCores(c.Resources.Requests.Cpu()),
			CpuCoresLimit:   convertQuantityToCores(c.Resources.Limits.Cpu()),
			MemoryMbRequest: convertQuantityToMb(c.Resources.Requests.Memory()),
			MemoryMbLimit:   convertQuantityToMb(c.Resources.Limits.Memory()),
		}
	}
}

func (w *DeploymentWrap) populateVolumesAndSecrets(podSpec v1.PodSpec) {
	volumeSourceMap := w.getVolumeSourceMap(podSpec)
	for i, c := range podSpec.Containers {
		for _, v := range c.VolumeMounts {
			sourceVolume, ok := volumeSourceMap[v.Name]
			if !ok {
				sourceVolume = &volumes.Unimplemented{}
			}
			if sourceVolume.Type() == "Secret" {
				w.Deployment.Containers[i].Secrets = append(w.Deployment.Containers[i].Secrets, &storage.EmbeddedSecret{
					Name: sourceVolume.Source(),
					Path: v.MountPath,
				})
				continue
			}
			w.Deployment.Containers[i].Volumes = append(w.Deployment.Containers[i].Volumes, &storage.Volume{
				Name:        v.Name,
				Source:      sourceVolume.Source(),
				Destination: v.MountPath,
				ReadOnly:    v.ReadOnly,
				Type:        sourceVolume.Type(),
			})
		}
	}
}

func (w *DeploymentWrap) populatePorts(podSpec v1.PodSpec) {
	for i, c := range podSpec.Containers {
		for _, p := range c.Ports {
			exposedPort := p.ContainerPort
			// If the port defines a host port, then it is exposed via that port instead of the container port
			if p.HostPort != 0 {
				exposedPort = p.HostPort
			}

			portConfig := &storage.PortConfig{
				Name:          p.Name,
				ContainerPort: p.ContainerPort,
				ExposedPort:   exposedPort,
				Protocol:      string(p.Protocol),
				Exposure:      storage.PortConfig_INTERNAL,
			}
			w.Deployment.Containers[i].Ports = append(w.Deployment.Containers[i].Ports, portConfig)
		}
	}
}

func (w *DeploymentWrap) toEvent(action central.ResourceAction) *central.SensorEvent {
	return &central.SensorEvent{
		Id:     w.GetId(),
		Action: action,
		Resource: &central.SensorEvent_Deployment{
			Deployment: w.Deployment,
		},
	}
}
