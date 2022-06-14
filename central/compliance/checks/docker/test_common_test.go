package docker

import (
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/docker/types"
	"github.com/stackrox/stackrox/pkg/uuid"
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
