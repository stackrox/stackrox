package imageutil

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/sensor/common/registry"
)

// IsInternalImage determines if the image represented by the given name
// is an "internal" image. An internal image is one which is hosted by an internal registry.
// An internal registry is on which is only accessible from within the cluster in which it lives.
// IsInternalImage uses the data from the given store to determine this.
// If registryStore is nil, registry.Singleton() is used as the store.
func IsInternalImage(image *storage.ImageName, registryStore *registry.Store) bool {
	// If the Sensor knows about the registry in which the image is hosted,
	// then the image must be "internal" to the cluster, as Sensor only tracks
	// "internal" registries.
	var store *registry.Store
	if registryStore != nil {
		store = registryStore
	} else {
		store = registry.Singleton()
	}
	reg, err := store.GetRegistryForImage(image)
	return reg != nil && err == nil
}
