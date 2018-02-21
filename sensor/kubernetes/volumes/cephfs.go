package volumes

import (
	"k8s.io/api/core/v1"
)

const cephFSType = "CephFS"

type cephfs struct {
	*v1.CephFSVolumeSource
}

// Source returns the source of the specific implementation
func (h *cephfs) Source() string {
	return h.Path
}

func (h *cephfs) Type() string {
	return cephFSType
}

func createCephfs(i interface{}) VolumeSource {
	cephVolume, ok := i.(*v1.CephFSVolumeSource)
	if !ok {
		return &Unimplemented{}
	}
	return &cephfs{
		CephFSVolumeSource: cephVolume,
	}
}

func init() {
	VolumeRegistry[cephFSType] = createCephfs
}
