package containerid

import (
	"github.com/stackrox/rox/generated/storage"
)

const (
	collectorContainerIDLength = 12
)

// ShortContainerIDFromInstance returns a short container id from the given container instance. Container IDs that are
// shorter than the length of a truncated container ID (12 hexadecimal characters) are returned without modification.
// This is intended to facilitate testing, which often uses fake container IDs.
func ShortContainerIDFromInstance(instance *storage.ContainerInstance) string {
	return ShortContainerIDFromInstanceID(instance.GetInstanceId().GetId())
}

// ShortContainerIDFromInstanceID returns a short container id from the given container instance ID.
// Container IDs that are shorter than the length of a truncated container ID (12 hexadecimal characters)
// are returned without modification.
func ShortContainerIDFromInstanceID(id string) string {
	if len(id) > collectorContainerIDLength {
		id = id[:collectorContainerIDLength]
	}
	return id
}
