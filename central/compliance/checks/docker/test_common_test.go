package docker

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/docker/types"
	"github.com/stackrox/rox/pkg/uuid"
)

func createTestNodes(names ...string) []*storage.Node {
	result := make([]*storage.Node, 0, len(names))
	for _, name := range names {
		result = append(result, &storage.Node{
			Id:   name,
			Name: name,
			ContainerRuntime: &storage.ContainerRuntimeInfo{
				Type:    storage.ContainerRuntime_DOCKER_CONTAINER_RUNTIME,
				Version: "1.2.3",
			},
		})
	}

	return result
}

func createTestPod(container *types.ContainerJSON) []*storage.Pod {
	if container == nil || container.ContainerJSONBase == nil {
		return nil
	}
	return []*storage.Pod{
		{
			Id:   uuid.NewV4().String(),
			Name: "test",
			LiveInstances: []*storage.ContainerInstance{
				{
					InstanceId: &storage.ContainerInstanceID{
						Id: container.ID,
					},
				},
			},
		},
	}
}
