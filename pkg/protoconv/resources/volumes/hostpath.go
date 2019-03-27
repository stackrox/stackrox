package volumes

import (
	v1 "k8s.io/api/core/v1"
)

const hostPathType = "HostPath"

type hostPath struct {
	*v1.HostPathVolumeSource
}

func (h *hostPath) Source() string {
	return h.Path
}

func (h *hostPath) Type() string {
	return hostPathType
}

func createHostPath(i interface{}) VolumeSource {
	hostPathVolume, ok := i.(*v1.HostPathVolumeSource)
	if !ok {
		return &Unimplemented{}
	}
	return &hostPath{
		HostPathVolumeSource: hostPathVolume,
	}
}

func init() {
	VolumeRegistry[hostPathType] = createHostPath
}
