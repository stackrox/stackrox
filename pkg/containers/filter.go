package containers

import "github.com/stackrox/rox/generated/storage"

// FilterRegularContainers returns only the containers with type REGULAR,
// filtering out init and ephemeral containers.
func FilterRegularContainers(containers []*storage.Container) []*storage.Container {
	regular := make([]*storage.Container, 0, len(containers))
	for _, c := range containers {
		if c.GetType() == storage.ContainerType_REGULAR {
			regular = append(regular, c)
		}
	}
	return regular
}
