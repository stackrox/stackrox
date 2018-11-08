package containerid

import (
	"github.com/stackrox/rox/generated/api/v1"
)

const (
	collectorContainerIDLength = 12
)

// ShortContainerIDFromInstance returns a short container id from the given container instance.
// It returns an empty string if the container id doesn't exist or is too short.
func ShortContainerIDFromInstance(instance *v1.ContainerInstance) string {
	id := instance.GetInstanceId().GetId()
	if len(id) < collectorContainerIDLength {
		return ""
	}
	return id[:collectorContainerIDLength]
}
