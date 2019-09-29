package docker

import "github.com/stackrox/rox/generated/storage"

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
