package listener

import (
	"reflect"

	pkgV1 "bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/env"
	"bitbucket.org/stack-rox/apollo/pkg/images"
	"github.com/golang/protobuf/ptypes"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var clusterID = env.ClusterID.Setting()

type wrap struct {
	*pkgV1.DeploymentEvent
}

func newDeploymentEventFromResource(obj interface{}, action pkgV1.ResourceAction, metaFieldIndex []int, resourceType string) (event *pkgV1.DeploymentEvent) {
	objValue := reflect.Indirect(reflect.ValueOf(obj))
	meta, ok := objValue.FieldByIndex(metaFieldIndex).Interface().(metav1.ObjectMeta)
	if !ok {
		logger.Errorf("obj %+v does not have an ObjectMeta field of the correct type", obj)
		return
	}

	// Ignore resources that are owned by another resource.
	if len(meta.OwnerReferences) > 0 {
		return
	}

	r := newWrap(meta, action, resourceType)

	r.populateFields(objValue)

	return r.DeploymentEvent
}

func newWrap(meta metav1.ObjectMeta, action pkgV1.ResourceAction, resourceType string) wrap {
	updatedTime, err := ptypes.TimestampProto(meta.CreationTimestamp.Time)
	if err != nil {
		logger.Error(err)
	}

	return wrap{
		&pkgV1.DeploymentEvent{
			Deployment: &pkgV1.Deployment{
				Id:        string(meta.UID),
				Name:      meta.Name,
				Type:      resourceType,
				Version:   meta.ResourceVersion,
				Namespace: meta.Namespace,
				UpdatedAt: updatedTime,
			},
			Action:    action,
			ClusterId: clusterID,
		},
	}
}

func (w *wrap) populateFields(objValue reflect.Value) {
	spec := objValue.FieldByName("Spec")
	if reflect.DeepEqual(spec, reflect.Value{}) {
		logger.Errorf("Obj %+v does not have a Spec field", objValue)
		return
	}

	w.populateReplicas(spec)

	podTemplate, ok := spec.FieldByName("Template").Interface().(v1.PodTemplateSpec)
	if !ok {
		logger.Errorf("Spec obj %+v does not have a Template field", spec)
		return
	}

	w.populateContainers(podTemplate.Spec)
}

func (w *wrap) populateContainers(podSpec v1.PodSpec) {
	w.Deployment.Containers = make([]*pkgV1.Container, len(podSpec.Containers))
	for i := range w.Deployment.Containers {
		w.Deployment.Containers[i] = new(pkgV1.Container)
	}

	w.populateContainerConfigs(podSpec)
	w.populateImages(podSpec)
	w.populateSecurityContext(podSpec)
	w.populateVolumes(podSpec)
	w.populatePorts(podSpec)
}

func (w *wrap) populateReplicas(spec reflect.Value) {
	replicaField := spec.FieldByName("Replicas")
	if reflect.DeepEqual(replicaField, reflect.Value{}) {
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

func (w *wrap) populateContainerConfigs(podSpec v1.PodSpec) {
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

		w.Deployment.Containers[i].Config = config
	}
}

func (w *wrap) populateImages(podSpec v1.PodSpec) {
	for i, c := range podSpec.Containers {
		w.Deployment.Containers[i].Image = images.GenerateImageFromString(c.Image)
	}
}

func (w *wrap) populateSecurityContext(podSpec v1.PodSpec) {
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

func (w *wrap) populateVolumes(podSpec v1.PodSpec) {
	volumeTypeMap := make(map[string]string)
	for _, v := range podSpec.Volumes {
		val := reflect.ValueOf(v.VolumeSource)

		for i := 0; i < val.NumField(); i++ {
			f := val.Field(i)
			if !f.IsNil() {
				volumeTypeMap[v.Name] = val.Type().Field(i).Name
			}
		}
	}

	for i, c := range podSpec.Containers {
		for _, v := range c.VolumeMounts {
			w.Deployment.Containers[i].Volumes = append(w.Deployment.Containers[i].Volumes, &pkgV1.Volume{
				Name:     v.Name,
				Path:     v.MountPath,
				ReadOnly: v.ReadOnly,
				Type:     volumeTypeMap[v.Name],
			})
		}
	}
}

func (w *wrap) populatePorts(podSpec v1.PodSpec) {
	for i, c := range podSpec.Containers {
		for _, p := range c.Ports {
			w.Deployment.Containers[i].Ports = append(w.Deployment.Containers[i].Ports, &pkgV1.PortConfig{
				Name:          p.Name,
				ContainerPort: p.ContainerPort,
				Protocol:      string(p.Protocol),
			})
		}
	}
}
