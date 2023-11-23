package resources

import (
	"encoding/json"
	"fmt"
	"reflect"

	ptypes "github.com/gogo/protobuf/types"
	openshiftAppsV1 "github.com/openshift/api/apps/v1"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	imageUtils "github.com/stackrox/rox/pkg/images/utils"
	"github.com/stackrox/rox/pkg/kubernetes"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/protoconv/k8s"
	"github.com/stackrox/rox/pkg/protoconv/resources/volumes"
	"github.com/stackrox/rox/pkg/stringutils"
	"github.com/stackrox/rox/pkg/timestamp"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/pkg/uuid"
	batchV1 "k8s.io/api/batch/v1"
	batchV1beta1 "k8s.io/api/batch/v1beta1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	openshiftEncodedDeploymentConfigAnnotation = `openshift.io/encoded-deployment-config`
	appArmorAnnotationTemplate                 = `container.apparmor.security.beta.kubernetes.io/%s`
)

var (
	log = logging.LoggerForModule()
)

// DeploymentWrap is a wrapper around a deployment to help convert the static fields
type DeploymentWrap struct {
	*storage.Deployment
	registryOverride string
}

// NewDeploymentWrap creates a deployment wrap from an existing deployment
func NewDeploymentWrap(d *storage.Deployment, registryOverride string) *DeploymentWrap {
	return &DeploymentWrap{
		Deployment:       d,
		registryOverride: registryOverride,
	}
}

// This checks if a reflect value is a Zero value, which means the field did not exist
func doesFieldExist(value reflect.Value) bool {
	return value.IsValid()
}

// IsTrackedOwnerReference validates the object is one that we are tracking as a Deployment
func IsTrackedOwnerReference(reference metav1.OwnerReference) bool {
	return kubernetes.IsDeploymentResource(reference.Kind) && kubernetes.IsNativeAPI(reference.APIVersion)
}

// NewDeploymentFromStaticResource returns a storage.Deployment from a k8s object
func NewDeploymentFromStaticResource(obj interface{}, deploymentType, clusterID, registryOverride string) (*storage.Deployment, error) {
	objMeta, err := meta.Accessor(obj)
	if err != nil {
		return nil, errors.Wrapf(err, "could not access metadata of object of type %T", obj)
	}
	kind := deploymentType

	// Ignore resources that are owned by another tracked resource.
	for _, ref := range objMeta.GetOwnerReferences() {
		if IsTrackedOwnerReference(ref) {
			return nil, nil
		}
	}

	// This only applies to OpenShift
	if encDeploymentConfig, ok := objMeta.GetLabels()[openshiftEncodedDeploymentConfigAnnotation]; ok {
		newMeta, newKind, err := extractDeploymentConfig(encDeploymentConfig)
		if err != nil {
			log.Error(err)
		} else {
			objMeta, kind = newMeta, newKind
		}
	}

	wrap := newWrap(objMeta, kind, clusterID, registryOverride)
	wrap.populateFields(obj)

	// Deployment ID is empty because the processed yaml comes from roxctl and therefore doesn't
	// get a  Kubernetes generated ID. This is a temporary ID only required for roxctl to distinguish
	// between different generated deployments.

	if wrap.Deployment.GetId() == "" {
		wrap.Deployment.Id = uuid.NewV4().String()
	}
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

func newWrap(meta metav1.Object, kind, clusterID, registryOverride string) *DeploymentWrap {
	createdTime, err := ptypes.TimestampProto(meta.GetCreationTimestamp().Time)
	if err != nil {
		log.Error(err)
	}
	return &DeploymentWrap{
		registryOverride: registryOverride,
		Deployment: &storage.Deployment{
			Id:             string(meta.GetUID()),
			ClusterId:      clusterID,
			Name:           meta.GetName(),
			Type:           kind,
			Namespace:      stringutils.OrDefault(meta.GetNamespace(), "default"),
			Labels:         meta.GetLabels(),
			Annotations:    meta.GetAnnotations(),
			Created:        createdTime,
			StateTimestamp: int64(timestamp.Now()),
		},
	}
}

// SpecToPodTemplateSpec turns a top level spec into a podTemplateSpec
func SpecToPodTemplateSpec(spec reflect.Value) (v1.PodTemplateSpec, error) {
	templateInterface := spec.FieldByName("Template")
	if !doesFieldExist(templateInterface) {
		return v1.PodTemplateSpec{}, errors.Errorf("obj %+v does not have a Template field", spec)
	}
	if templateInterface.Type().Kind() == reflect.Ptr && !templateInterface.IsNil() {
		templateInterface = templateInterface.Elem()
	}
	podTemplate, ok := templateInterface.Interface().(v1.PodTemplateSpec)
	if !ok {
		return v1.PodTemplateSpec{}, errors.New("not a valid PodTemplateSpec")
	}
	return podTemplate, nil
}

func getMountPropagation(mountPropagation *v1.MountPropagationMode) storage.Volume_MountPropagation {
	if mountPropagation == nil {
		return storage.Volume_NONE
	}

	switch *mountPropagation {
	case v1.MountPropagationHostToContainer:
		return storage.Volume_HOST_TO_CONTAINER
	case v1.MountPropagationBidirectional:
		return storage.Volume_BIDIRECTIONAL
	default:
		return storage.Volume_NONE
	}
}

func getSeccompProfileType(profileType v1.SeccompProfileType) storage.SecurityContext_SeccompProfile_ProfileType {
	switch profileType {
	case v1.SeccompProfileTypeUnconfined:
		return storage.SecurityContext_SeccompProfile_UNCONFINED
	case v1.SeccompProfileTypeLocalhost:
		return storage.SecurityContext_SeccompProfile_LOCALHOST
	default:
		return storage.SecurityContext_SeccompProfile_RUNTIME_DEFAULT
	}
}

func makeSeccompProfileWithDefaults(s *v1.SecurityContext, podSec *v1.PodSecurityContext) *storage.SecurityContext_SeccompProfile {
	if s != nil {
		if profile := convertSeccompProfile(s.SeccompProfile); profile != nil {
			return profile
		}
	}

	if podSec != nil {
		return convertSeccompProfile(podSec.SeccompProfile)
	}

	return nil
}

func convertSeccompProfile(sp *v1.SeccompProfile) *storage.SecurityContext_SeccompProfile {
	if sp == nil {
		return nil
	}
	seccompProfile := &storage.SecurityContext_SeccompProfile{
		Type: getSeccompProfileType(sp.Type),
	}
	if sp.LocalhostProfile != nil {
		seccompProfile.LocalhostProfile = *sp.LocalhostProfile
	}
	return seccompProfile
}

func makeSELinuxWithDefaults(s *v1.SecurityContext, podSec *v1.PodSecurityContext) *storage.SecurityContext_SELinux {
	if s != nil {
		if sel := convertSELinux(s.SELinuxOptions); sel != nil {
			return sel
		}
	}

	if podSec != nil {
		return convertSELinux(podSec.SELinuxOptions)
	}

	return nil
}

func convertSELinux(SELinux *v1.SELinuxOptions) *storage.SecurityContext_SELinux {
	if SELinux == nil {
		return nil
	}

	return &storage.SecurityContext_SELinux{
		User:  SELinux.User,
		Role:  SELinux.Role,
		Type:  SELinux.Type,
		Level: SELinux.Level,
	}
}

func (w *DeploymentWrap) populateFields(obj interface{}) {
	objValue := reflect.Indirect(reflect.ValueOf(obj))
	spec := objValue.FieldByName("Spec")
	if !doesFieldExist(spec) {
		log.Errorf("Obj %+v does not have a Spec field", objValue)
		return
	}

	w.populateReplicas(spec, obj)

	var podSpec v1.PodSpec

	switch o := obj.(type) {
	case *openshiftAppsV1.DeploymentConfig:
		if o.Spec.Template == nil {
			log.Errorf("Spec obj %+v does not have a Template field or is not a pointer pod spec", spec)
			return
		}
		podSpec = o.Spec.Template.Spec
		// Pods don't have the abstractions that higher level objects have so maintain it's lifecycle independently
	case *v1.Pod:
		// Standalone Pods do not have a PodTemplate, like the other deployment
		// types do. So, we need to directly access the Pod's Spec field,
		// instead of looking for it inside a PodTemplate.
		podSpec = o.Spec
	// batch/v1beta1 CronJob is deprecated in v1.21+, unavailable in v1.25+.
	case *batchV1beta1.CronJob:
		podSpec = o.Spec.JobTemplate.Spec.Template.Spec
	case *batchV1.CronJob:
		podSpec = o.Spec.JobTemplate.Spec.Template.Spec
	default:
		podTemplate, err := SpecToPodTemplateSpec(spec)
		if err != nil {
			utils.Should(errors.Wrapf(err, "spec obj %+v cannot be converted to a pod template spec", spec))
			return
		}
		podSpec = podTemplate.Spec
	}

	w.PopulateDeploymentFromPodSpec(podSpec)
}

// PopulateDeploymentFromPodSpec fills in the initialized wrap with data from the passed pod spec
func (w *DeploymentWrap) PopulateDeploymentFromPodSpec(podSpec v1.PodSpec) {
	w.HostNetwork = podSpec.HostNetwork
	w.HostPid = podSpec.HostPID
	w.HostIpc = podSpec.HostIPC
	w.RuntimeClass = stringutils.PointerOrDefault(podSpec.RuntimeClassName, "")
	w.populateTolerations(podSpec)
	w.populateServiceAccount(podSpec)
	w.populateAutomountServiceAccountToken(podSpec)
	w.populateImagePullSecrets(podSpec)

	w.populateContainers(podSpec)
}

func (w *DeploymentWrap) populateTolerations(podSpec v1.PodSpec) {
	w.Tolerations = make([]*storage.Toleration, 0, len(podSpec.Tolerations))
	for _, toleration := range podSpec.Tolerations {
		w.Tolerations = append(w.Tolerations, &storage.Toleration{
			Key:         toleration.Key,
			Value:       toleration.Value,
			Operator:    k8s.ToRoxTolerationOperator(toleration.Operator),
			TaintEffect: k8s.ToRoxTaintEffect(toleration.Effect),
		})
	}
}

func (w *DeploymentWrap) populateContainers(podSpec v1.PodSpec) {
	w.Deployment.Containers = make([]*storage.Container, 0, len(podSpec.Containers))
	for _, c := range podSpec.Containers {
		w.Deployment.Containers = append(w.Deployment.Containers, &storage.Container{
			Id:   fmt.Sprintf("%s:%s", w.Deployment.Id, c.Name),
			Name: c.Name,
		})
	}

	w.populateContainerConfigs(podSpec)
	w.populateImages(podSpec)
	w.populateSecurityContext(podSpec)
	w.populateVolumesAndSecrets(podSpec)
	w.populatePorts(podSpec)
	w.populateResources(podSpec)
	w.populateProbes(podSpec)
}

func (w *DeploymentWrap) populateServiceAccount(podSpec v1.PodSpec) {
	w.ServiceAccount = stringutils.OrDefault(podSpec.ServiceAccountName, "default")
}

func (w *DeploymentWrap) populateAutomountServiceAccountToken(podSpec v1.PodSpec) {
	if podSpec.AutomountServiceAccountToken == nil {
		w.AutomountServiceAccountToken = true
	} else {
		w.AutomountServiceAccountToken = *podSpec.AutomountServiceAccountToken
	}
}

func (w *DeploymentWrap) populateImagePullSecrets(podSpec v1.PodSpec) {
	secrets := make([]string, 0, len(podSpec.ImagePullSecrets))
	for _, s := range podSpec.ImagePullSecrets {
		secrets = append(secrets, s.Name)
	}
	w.ImagePullSecrets = secrets
}

func (w *DeploymentWrap) populateDaemonSetReplicaSet(obj interface{}) {
	ds := reflect.ValueOf(obj)
	if ds.Kind() == reflect.Ptr {
		ds = ds.Elem()
	}
	status := ds.FieldByName("Status")
	if !doesFieldExist(status) {
		return
	}
	na := status.FieldByName("NumberAvailable")
	if !doesFieldExist(na) {
		return
	}
	w.Replicas = na.Int()
}

func (w *DeploymentWrap) populateReplicas(spec reflect.Value, obj interface{}) {
	if w.Deployment.GetType() == kubernetes.DaemonSet {
		w.populateDaemonSetReplicaSet(obj)
		return
	}

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

func (w *DeploymentWrap) populateContainerConfigs(podSpec v1.PodSpec) {
	for i, c := range podSpec.Containers {
		config := &storage.ContainerConfig{
			Command:   c.Command,
			Args:      c.Args,
			Directory: c.WorkingDir,
		}

		envSlice := make([]*storage.ContainerConfig_EnvironmentConfig, len(c.Env))
		for i, env := range c.Env {
			if env.ValueFrom == nil {
				envSlice[i] = &storage.ContainerConfig_EnvironmentConfig{
					Key:          env.Name,
					Value:        env.Value,
					EnvVarSource: storage.ContainerConfig_EnvironmentConfig_RAW,
				}
			} else {
				var value string
				var envVarSrc storage.ContainerConfig_EnvironmentConfig_EnvVarSource
				switch {
				case env.ValueFrom.SecretKeyRef != nil:
					envVarSrc = storage.ContainerConfig_EnvironmentConfig_SECRET_KEY
					ref := env.ValueFrom.SecretKeyRef
					value = fmt.Sprintf("Refers to secret %q with key %q", ref.Name, ref.Key)
				case env.ValueFrom.ConfigMapKeyRef != nil:
					envVarSrc = storage.ContainerConfig_EnvironmentConfig_CONFIG_MAP_KEY
					ref := env.ValueFrom.ConfigMapKeyRef
					value = fmt.Sprintf("Refers to config map %q with key %q", ref.Name, ref.Key)
				case env.ValueFrom.FieldRef != nil:
					envVarSrc = storage.ContainerConfig_EnvironmentConfig_FIELD
					ref := env.ValueFrom.FieldRef
					value = fmt.Sprintf("Refers to field %q", ref.FieldPath)
				case env.ValueFrom.ResourceFieldRef != nil:
					envVarSrc = storage.ContainerConfig_EnvironmentConfig_RESOURCE_FIELD
					ref := env.ValueFrom.ResourceFieldRef
					value = fmt.Sprintf("Refers to resource %q from container %q", ref.Resource, ref.ContainerName)
				default:
					envVarSrc = storage.ContainerConfig_EnvironmentConfig_UNKNOWN
					value = "Unknown environment value reference"
				}
				envSlice[i] = &storage.ContainerConfig_EnvironmentConfig{
					Key:          env.Name,
					Value:        value,
					EnvVarSource: envVarSrc,
				}
			}
		}

		config.Env = envSlice

		if s := c.SecurityContext; s != nil {
			if uid := s.RunAsUser; uid != nil {
				config.Uid = *uid
			}
		}

		appArmorAnnotation := fmt.Sprintf(appArmorAnnotationTemplate, c.Name)
		appArmorProfile := w.Annotations[appArmorAnnotation]
		config.AppArmorProfile = appArmorProfile

		w.Deployment.Containers[i].Config = config
	}
}

func (w *DeploymentWrap) populateImages(podSpec v1.PodSpec) {
	for i, c := range podSpec.Containers {
		parsedImage, err := imageUtils.GenerateImageFromStringWithOverride(c.Image, w.registryOverride)

		if err != nil {
			log.Error(err)
			parsedImage = &storage.ContainerImage{
				Name: &storage.ImageName{
					FullName: fmt.Sprintf("%s is an invalid image", c.Image),
				},
			}
		}
		w.Deployment.Containers[i].Image = parsedImage
	}
}

func (w *DeploymentWrap) populateSecurityContext(podSpec v1.PodSpec) {
	for i, c := range podSpec.Containers {
		sc := &storage.SecurityContext{}
		s := c.SecurityContext
		if s != nil {
			if p := s.Privileged; p != nil {
				sc.Privileged = *p
			}

			if p := s.ReadOnlyRootFilesystem; p != nil {
				sc.ReadOnlyRootFilesystem = *p
			}

			if capabilities := s.Capabilities; capabilities != nil {
				for _, add := range capabilities.Add {
					sc.AddCapabilities = append(sc.AddCapabilities, string(add))
				}

				for _, drop := range capabilities.Drop {
					sc.DropCapabilities = append(sc.DropCapabilities, string(drop))
				}
			}

			if ape := s.AllowPrivilegeEscalation; ape != nil {
				sc.AllowPrivilegeEscalation = *ape
			}
		}
		sc.Selinux = makeSELinuxWithDefaults(s, podSpec.SecurityContext)
		sc.SeccompProfile = makeSeccompProfileWithDefaults(s, podSpec.SecurityContext)

		w.Deployment.Containers[i].SecurityContext = sc
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

func (w *DeploymentWrap) populateResources(podSpec v1.PodSpec) {
	for i, c := range podSpec.Containers {
		w.Deployment.Containers[i].Resources = &storage.Resources{
			CpuCoresRequest: k8s.ConvertQuantityToCores(c.Resources.Requests.Cpu()),
			CpuCoresLimit:   k8s.ConvertQuantityToCores(c.Resources.Limits.Cpu()),
			MemoryMbRequest: k8s.ConvertQuantityToMB(c.Resources.Requests.Memory()),
			MemoryMbLimit:   k8s.ConvertQuantityToMB(c.Resources.Limits.Memory()),
		}
	}
}

// Populates volumes and secrets that are referenced by both volumes and environment variables.
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
				Name:             v.Name,
				Source:           sourceVolume.Source(),
				Destination:      v.MountPath,
				ReadOnly:         v.ReadOnly,
				Type:             sourceVolume.Type(),
				MountPropagation: getMountPropagation(v.MountPropagation),
			})
		}

		for _, s := range podSpec.ImagePullSecrets {
			w.Deployment.Containers[i].Secrets = append(w.Deployment.Containers[i].Secrets, &storage.EmbeddedSecret{
				Name: s.Name,
			})
		}

		for _, e := range c.EnvFrom {
			if e.SecretRef != nil {
				w.Deployment.Containers[i].Secrets = append(w.Deployment.Containers[i].Secrets, &storage.EmbeddedSecret{
					Name: e.SecretRef.Name,
				})
			}
		}

		for _, e := range c.Env {
			if e.ValueFrom != nil && e.ValueFrom.SecretKeyRef != nil {
				w.Deployment.Containers[i].Secrets = append(w.Deployment.Containers[i].Secrets, &storage.EmbeddedSecret{
					Name: e.ValueFrom.SecretKeyRef.Name,
				})
			}
		}
	}
}

func (w *DeploymentWrap) populatePorts(podSpec v1.PodSpec) {
	w.Ports = nil
	for i, c := range podSpec.Containers {
		for _, p := range c.Ports {
			var exposures []*storage.PortConfig_ExposureInfo
			exposureLevel := storage.PortConfig_UNSET
			if p.HostPort != 0 {
				hostPortExposure := &storage.PortConfig_ExposureInfo{
					Level:    storage.PortConfig_HOST,
					NodePort: p.HostPort,
				}
				exposures = []*storage.PortConfig_ExposureInfo{hostPortExposure}
				exposureLevel = storage.PortConfig_HOST
			}

			protocolStr := string(p.Protocol)
			if protocolStr == "" {
				protocolStr = string(v1.ProtocolTCP)
			}

			portConfig := &storage.PortConfig{
				Name:          p.Name,
				ContainerPort: p.ContainerPort,
				Protocol:      protocolStr,
				Exposure:      exposureLevel,
				ExposureInfos: exposures,
			}
			w.Deployment.Containers[i].Ports = append(w.Deployment.Containers[i].Ports, portConfig)
			w.Ports = append(w.Ports, portConfig)
		}
	}
}

func (w *DeploymentWrap) populateProbes(podSpec v1.PodSpec) {
	for i, c := range podSpec.Containers {
		if c.LivenessProbe == nil || *c.LivenessProbe == (v1.Probe{}) {
			w.Deployment.Containers[i].LivenessProbe = &storage.LivenessProbe{Defined: false}
		} else {
			w.Deployment.Containers[i].LivenessProbe = &storage.LivenessProbe{Defined: true}
		}

		if c.ReadinessProbe == nil || *c.ReadinessProbe == (v1.Probe{}) {
			w.Deployment.Containers[i].ReadinessProbe = &storage.ReadinessProbe{Defined: false}
		} else {
			w.Deployment.Containers[i].ReadinessProbe = &storage.ReadinessProbe{Defined: true}
		}
	}
}
