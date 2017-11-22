package swarm

import (
	"bitbucket.org/stack-rox/apollo/apollo/types"
	"github.com/docker/docker/api/types/swarm"
)

// Container is the Swarm specific implementation of the Container Interface
type swarmToContainer struct {
	service swarm.Service
}

// ConvertToContainer converts a swarm service to a generic "container"
func (s swarmToContainer) ConvertToContainer() *types.Container {
	return &types.Container{
		ID:         s.service.ID,
		Name:       s.service.Spec.Name,
		Image:      types.GenerateImageFromString(s.service.Spec.TaskTemplate.ContainerSpec.Image),
		Privileged: false,
	}
}
