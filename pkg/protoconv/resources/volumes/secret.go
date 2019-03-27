package volumes

import (
	v1 "k8s.io/api/core/v1"
)

const secretType = "Secret"

type secret struct {
	*v1.SecretVolumeSource
}

func (h *secret) Source() string {
	return h.SecretName
}

func (h *secret) Type() string {
	return secretType
}

func createSecret(i interface{}) VolumeSource {
	secretVolume, ok := i.(*v1.SecretVolumeSource)
	if !ok {
		return &Unimplemented{}
	}
	return &secret{
		SecretVolumeSource: secretVolume,
	}
}

func init() {
	VolumeRegistry[secretType] = createSecret
}
