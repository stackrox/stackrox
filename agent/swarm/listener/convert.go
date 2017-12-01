package listener

import (
	"fmt"

	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/images"
	"github.com/docker/docker/api/types/swarm"
	"github.com/golang/protobuf/ptypes"
)

type serviceWrap swarm.Service

func (s serviceWrap) asDeployment() *v1.Deployment {
	updatedTime, err := ptypes.TimestampProto(s.UpdateStatus.CompletedAt)
	if err != nil {
		log.Error(err)
	}

	return &v1.Deployment{
		Id:        s.ID,
		Name:      s.Spec.Name,
		Version:   fmt.Sprintf("%d", s.Version.Index),
		Type:      modeWrap(s.Spec.Mode).asType(),
		UpdatedAt: updatedTime,
		Image:     images.GenerateImageFromString(s.Spec.TaskTemplate.ContainerSpec.Image),
	}
}

type modeWrap swarm.ServiceMode

func (m modeWrap) asType() string {
	if m.Replicated != nil {
		return `Replicated`
	}

	return `Global`
}
