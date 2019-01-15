package volumes

import (
	"k8s.io/api/core/v1"
)

const glusterfsType = "Glusterfs"

type glusterfs struct {
	*v1.GlusterfsVolumeSource
}

func (h *glusterfs) Source() string {
	return h.Path
}

func (h *glusterfs) Type() string {
	return glusterfsType
}

func createGlusterfs(i interface{}) VolumeSource {
	glusterVolume, ok := i.(*v1.GlusterfsVolumeSource)
	if !ok {
		return &Unimplemented{}
	}
	return &glusterfs{
		GlusterfsVolumeSource: glusterVolume,
	}
}

func init() {
	VolumeRegistry[glusterfsType] = createGlusterfs
}
