package volumes

import (
	"k8s.io/api/core/v1"
)

const rbdType = "RBD"

type rbd struct {
	*v1.RBDVolumeSource
}

func (h *rbd) Source() string {
	return h.RBDPool + "/" + h.RBDImage
}

func (h *rbd) Type() string {
	return rbdType
}

func createRBD(i interface{}) VolumeSource {
	rbdVolume, ok := i.(*v1.RBDVolumeSource)
	if !ok {
		return &Unimplemented{}
	}
	return &rbd{
		RBDVolumeSource: rbdVolume,
	}
}

func init() {
	VolumeRegistry[rbdType] = createRBD
}
