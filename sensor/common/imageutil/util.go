package imageutil

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/sensor/common/registry"
)

var (
	log = logging.LoggerForModule()
)

// IsInternalImage determines if the image represented by the given name
// is an "internal" image. An internal image is one which is hosted by an internal registry.
// An internal registry is on which is only accessible from within the cluster in which it lives.
func IsInternalImage(image *storage.ImageName) bool {
	// If the Sensor knows about the registry in which the image is hosted,
	// then the image must be "internal" to the cluster, as Sensor only tracks
	// "internal" registries.
	reg, err := registry.Singleton().GetRegistryForImage(image)
	if err != nil {
		log.Infof("Registry %q for image %s is unknown at this time", image.GetRegistry(), image.GetFullName())
		return false
	}

	return reg != nil
}
