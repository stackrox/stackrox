package volumes

import (
	v1 "k8s.io/api/core/v1"
)

const persistentVolumeClaimType = "PersistentVolumeClaim"

type persistentVolumeClaim struct {
	*v1.PersistentVolumeClaimVolumeSource
}

func (h *persistentVolumeClaim) Source() string {
	return h.ClaimName
}

func (h *persistentVolumeClaim) Type() string {
	return persistentVolumeClaimType
}

func createPersistentVolumeClaim(i interface{}) VolumeSource {
	persistent, ok := i.(*v1.PersistentVolumeClaimVolumeSource)
	if !ok {
		return &Unimplemented{}
	}
	return &persistentVolumeClaim{
		PersistentVolumeClaimVolumeSource: persistent,
	}
}

func init() {
	VolumeRegistry[persistentVolumeClaimType] = createPersistentVolumeClaim
}
