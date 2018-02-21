package volumes

import (
	"k8s.io/api/core/v1"
)

const configMapType = "ConfigMap"

type configMap struct {
	*v1.ConfigMapVolumeSource
}

func (h *configMap) Source() string {
	return h.Name
}

func (h *configMap) Type() string {
	return configMapType
}

func createConfigMap(i interface{}) VolumeSource {
	configVolume, ok := i.(*v1.ConfigMapVolumeSource)
	if !ok {
		return &Unimplemented{}
	}
	return &configMap{
		ConfigMapVolumeSource: configVolume,
	}
}

func init() {
	VolumeRegistry[configMapType] = createConfigMap
}
