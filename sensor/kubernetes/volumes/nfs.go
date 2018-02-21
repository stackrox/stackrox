package volumes

import (
	"k8s.io/api/core/v1"
)

const nfsType = "NFS"

type nfs struct {
	*v1.NFSVolumeSource
}

func (h *nfs) Source() string {
	return h.Server + "/" + h.Path
}

func (h *nfs) Type() string {
	return nfsType
}

func createNFS(i interface{}) VolumeSource {
	nfsVolume, ok := i.(*v1.NFSVolumeSource)
	if !ok {
		return &Unimplemented{}
	}
	return &nfs{
		NFSVolumeSource: nfsVolume,
	}
}

func init() {
	VolumeRegistry[nfsType] = createNFS
}
