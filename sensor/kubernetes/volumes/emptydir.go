package volumes

import "k8s.io/api/core/v1"

const emptyDirType = "EmptyDir"

type emptyDir struct {
	*v1.EmptyDirVolumeSource
}

func (e *emptyDir) Source() string {
	return string(e.Medium)
}

func (e *emptyDir) Type() string {
	return emptyDirType
}

func createEmptyDir(i interface{}) VolumeSource {
	emptyDirVolume, ok := i.(*v1.EmptyDirVolumeSource)
	if !ok {
		return &Unimplemented{}
	}
	return &emptyDir{
		EmptyDirVolumeSource: emptyDirVolume,
	}
}

func init() {
	VolumeRegistry[emptyDirType] = createEmptyDir
}
