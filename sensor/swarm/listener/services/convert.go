package services

import (
	"fmt"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/client"
	ptypes "github.com/gogo/protobuf/types"
	timestamp "github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/docker"
	imageTypes "github.com/stackrox/rox/pkg/images/types"
	imageUtils "github.com/stackrox/rox/pkg/images/utils"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/protoconv"
)

const (
	defaultNamespace = "default"

	nanoCPUS = 1000 * 1000 * 1000
	megabyte = 1024 * 1024
)

var log = logging.LoggerForModule()

type serviceWrap swarm.Service

func (s serviceWrap) getNetworkName(client *client.Client, id string) (string, error) {
	ctx, cancel := docker.TimeoutContext()
	defer cancel()
	network, err := client.NetworkInspect(ctx, id, types.NetworkInspectOptions{})
	if err != nil {
		return "", err
	}
	return network.Name, nil
}

func (s serviceWrap) asDeployment(client *client.Client, retryGetImageSha bool) *v1.Deployment {
	var updatedTime *timestamp.Timestamp
	up := s.UpdateStatus
	var err error
	if up != nil && up.CompletedAt != nil {
		updatedTime, err = ptypes.TimestampProto(*up.CompletedAt)
		if err != nil {
			log.Error(err)
		}
	} else {
		updatedTime, err = ptypes.TimestampProto(s.CreatedAt)
		if err != nil {
			log.Error(err)
		}
	}

	image := imageUtils.GenerateImageFromString(s.Spec.TaskTemplate.ContainerSpec.Image)

	if retryGetImageSha {
		retries := 0
		for image.GetName().GetSha() == "" && retries <= 15 {
			time.Sleep(time.Second)
			image.Name.Sha = s.getSHAFromTask(client)
			retries++
		}
		if image.GetName().GetSha() == "" {
			log.Warnf("Couldn't find an image SHA for service %s", s.ID)
		}
	}

	m := modeWrap(s.Spec.Mode)

	return &v1.Deployment{
		Id:        s.ID,
		Name:      s.Spec.Name,
		Namespace: defaultNamespace,
		Version:   fmt.Sprintf("%d", s.Version.Index),
		Type:      m.asType(),
		Replicas:  m.asReplica(),
		Labels:    protoconv.ConvertDeploymentKeyValueMap(s.Spec.Labels),
		UpdatedAt: updatedTime,
		Containers: []*v1.Container{
			{
				Config:          s.getContainerConfig(),
				Image:           image,
				SecurityContext: s.getSecurityContext(),
				Volumes:         s.getVolumes(),
				Ports:           s.getPorts(),
				Secrets:         s.getSecrets(),
				Resources:       s.getResources(),
				Id:              "c_" + s.ID,
			},
		},
	}
}

func convertNanoCPUsToCores(nanos int64) float32 {
	return float32(float64(nanos) / nanoCPUS)
}

func convertMemoryBytesToMb(bytes int64) float32 {
	return float32(float64(bytes) / megabyte)
}

func (s serviceWrap) getResources() *v1.Resources {
	resources := s.Spec.TaskTemplate.Resources
	if resources == nil {
		return nil
	}
	var v1Resources v1.Resources
	if resources.Limits != nil {
		v1Resources.CpuCoresLimit = convertNanoCPUsToCores(resources.Limits.NanoCPUs)
		v1Resources.MemoryMbLimit = convertMemoryBytesToMb(resources.Limits.MemoryBytes)
	}
	if resources.Reservations != nil {
		v1Resources.CpuCoresRequest = convertNanoCPUsToCores(resources.Reservations.NanoCPUs)
		v1Resources.MemoryMbRequest = convertMemoryBytesToMb(resources.Reservations.MemoryBytes)
	}
	return &v1Resources
}

func (s serviceWrap) getContainerConfig() *v1.ContainerConfig {
	spec := s.Spec.TaskTemplate.ContainerSpec

	envSlice := make([]*v1.ContainerConfig_EnvironmentConfig, 0, len(spec.Env))
	for _, env := range spec.Env {
		parts := strings.SplitN(env, `=`, 2)
		if len(parts) == 2 {
			envSlice = append(envSlice, &v1.ContainerConfig_EnvironmentConfig{
				Key:   parts[0],
				Value: parts[1],
			})
		}
	}

	return &v1.ContainerConfig{
		Args:      spec.Args,
		Command:   spec.Command,
		Directory: spec.Dir,
		Env:       envSlice,
		User:      spec.User,
	}
}

func (s serviceWrap) getSecurityContext() *v1.SecurityContext {
	spec := s.Spec.TaskTemplate.ContainerSpec

	if spec.Privileges == nil || spec.Privileges.SELinuxContext == nil {
		return nil
	}

	return &v1.SecurityContext{
		Selinux: &v1.SecurityContext_SELinux{
			User:  spec.Privileges.SELinuxContext.User,
			Role:  spec.Privileges.SELinuxContext.Role,
			Type:  spec.Privileges.SELinuxContext.Type,
			Level: spec.Privileges.SELinuxContext.Level,
		},
	}
}

func (s serviceWrap) getPorts() []*v1.PortConfig {
	output := make([]*v1.PortConfig, len(s.Endpoint.Ports))
	for i, p := range s.Endpoint.Ports {
		output[i] = &v1.PortConfig{
			Name:          p.Name,
			ExposedPort:   int32(p.PublishedPort),
			ContainerPort: int32(p.TargetPort),
			Protocol:      string(p.Protocol),
			Exposure:      isPublished(p),
		}
	}

	return output
}

func isPublished(port swarm.PortConfig) v1.PortConfig_Exposure {
	if port.PublishedPort == 0 {
		return v1.PortConfig_INTERNAL
	}
	if port.PublishMode == swarm.PortConfigPublishModeHost {
		return v1.PortConfig_NODE
	}
	return v1.PortConfig_EXTERNAL
}

func (s serviceWrap) getVolumes() []*v1.Volume {
	spec := s.Spec.TaskTemplate.ContainerSpec

	output := make([]*v1.Volume, len(spec.Mounts))

	for i, m := range spec.Mounts {
		output[i] = &v1.Volume{
			Name:        m.Source,
			Source:      m.Source,
			Destination: m.Target,
			Type:        string(m.Type),
			ReadOnly:    m.ReadOnly,
		}
	}
	return output
}

func (s serviceWrap) getSecrets() []*v1.EmbeddedSecret {
	spec := s.Spec.TaskTemplate.ContainerSpec
	secrets := make([]*v1.EmbeddedSecret, 0, len(spec.Secrets))
	for _, secret := range spec.Secrets {
		path := ""
		if secret.File != nil {
			path = `/run/secrets/` + secret.File.Name
		}
		secrets = append(secrets, &v1.EmbeddedSecret{
			Id:   secret.SecretID,
			Name: secret.SecretName,
			Path: path,
		})
	}
	return secrets
}

func (s serviceWrap) getSHAFromTask(client *client.Client) string {
	opts := filters.NewArgs()
	opts.Add("service", s.ID)
	opts.Add("desired-state", "running")
	ctx, cancel := docker.TimeoutContext()
	defer cancel()
	tasks, err := client.TaskList(ctx, types.TaskListOptions{Filters: opts})
	if err != nil {
		log.Errorf("Couldn't enumerate service %s tasks to get image SHA: %s", s.ID, err)
		return ""
	}
	for _, t := range tasks {
		id := t.Status.ContainerStatus.ContainerID
		if id == "" {
			continue
		}
		ctx, cancel := docker.TimeoutContext()
		defer cancel()
		container, err := client.ContainerInspect(ctx, id)
		if err != nil {
			log.Warnf("Couldn't inspect %s to get image SHA for service %s: %s", id, s.ID, err)
			continue
		}
		// TODO(cg): If the image is specified only as a tag, and Swarm can't
		// resolve to a SHA256 digest when launching, the image SHA may actually
		// differ between tasks on different nodes.
		return imageTypes.NewDigest(container.Image).Digest()
	}
	return ""
}

type modeWrap swarm.ServiceMode

func (m modeWrap) asType() string {
	if m.Replicated != nil {
		return `Replicated`
	}

	return `Global`
}

func (m modeWrap) asReplica() int64 {
	if m.Replicated != nil {
		return int64(*m.Replicated.Replicas)
	}

	return 0
}
