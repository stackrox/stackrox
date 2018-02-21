package volumes

import (
	"k8s.io/api/core/v1"
)

const cinderType = "Cinder"

type cinder struct {
	*v1.CinderVolumeSource
}

// Source returns the source of the specific implementation
func (h *cinder) Source() string {
	return h.VolumeID
}

func (h *cinder) Type() string {
	return cinderType
}

func createCinder(i interface{}) VolumeSource {
	cinderVolume, ok := i.(*v1.CinderVolumeSource)
	if !ok {
		return &Unimplemented{}
	}
	return &cinder{
		CinderVolumeSource: cinderVolume,
	}
}

func init() {
	VolumeRegistry[cinderType] = createCinder
}
