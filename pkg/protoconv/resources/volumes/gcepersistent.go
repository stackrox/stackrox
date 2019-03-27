package volumes

import v1 "k8s.io/api/core/v1"

const gcePersistentDiskType = "GCEPersistentDisk"

type gcePersistentDisk struct {
	*v1.GCEPersistentDiskVolumeSource
}

func (e *gcePersistentDisk) Source() string {
	return string(e.PDName)
}

func (e *gcePersistentDisk) Type() string {
	return gcePersistentDiskType
}

func createGCEPersistentDisk(i interface{}) VolumeSource {
	gceVolume, ok := i.(*v1.GCEPersistentDiskVolumeSource)
	if !ok {
		return &Unimplemented{}
	}
	return &gcePersistentDisk{
		GCEPersistentDiskVolumeSource: gceVolume,
	}
}

func init() {
	VolumeRegistry[gcePersistentDiskType] = createGCEPersistentDisk
}
