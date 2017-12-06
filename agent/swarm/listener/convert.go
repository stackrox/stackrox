package listener

import (
	"context"
	"fmt"

	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/images"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/client"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/timestamp"
)

type serviceWrap swarm.Service

func (s serviceWrap) asDeployment(client *client.Client) *v1.Deployment {
	var updatedTime *timestamp.Timestamp
	up := s.UpdateStatus
	if up != nil && up.CompletedAt != nil {
		var err error
		updatedTime, err = ptypes.TimestampProto(*up.CompletedAt)
		if err != nil {
			log.Error(err)
		}
	}

	image := images.GenerateImageFromString(s.Spec.TaskTemplate.ContainerSpec.Image)
	if image.Sha == "" {
		image.Sha = s.getSHAFromTask(client)
	}

	return &v1.Deployment{
		Id:        s.ID,
		Name:      s.Spec.Name,
		Version:   fmt.Sprintf("%d", s.Version.Index),
		Type:      modeWrap(s.Spec.Mode).asType(),
		UpdatedAt: updatedTime,
		Image:     image,
	}
}

func (s serviceWrap) getSHAFromTask(client *client.Client) string {
	opts := filters.NewArgs()
	opts.Add("service", s.ID)
	opts.Add("desired-state", "running")
	tasks, err := client.TaskList(context.Background(), types.TaskListOptions{Filters: opts})
	if err != nil {
		log.Errorf("Couldn't enumerate service %s tasks to get image SHA: %s", s.ID, err)
		return ""
	}
	for _, t := range tasks {
		id := t.Status.ContainerStatus.ContainerID
		if id == "" {
			continue
		}
		container, err := client.ContainerInspect(context.Background(), id)
		if err != nil {
			log.Warnf("Couldn't inspect %s to get image SHA for service %s: %s", id, s.ID, err)
			continue
		}
		// TODO(cg): If the image is specified only as a tag, and Swarm can't
		// resolve to a SHA256 digest when launching, the image SHA may actually
		// differ between tasks on different nodes.
		return container.Image
	}
	log.Warnf("Couldn't find an image SHA for service %s", s.ID)
	return ""
}

type modeWrap swarm.ServiceMode

func (m modeWrap) asType() string {
	if m.Replicated != nil {
		return `Replicated`
	}

	return `Global`
}
