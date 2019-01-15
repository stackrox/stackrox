package volumes

import (
	"k8s.io/api/core/v1"
)

const azureDiskType = "AzureDisk"

type azureDisk struct {
	*v1.AzureDiskVolumeSource
}

func (h *azureDisk) Source() string {
	return h.DiskName
}

func (h *azureDisk) Type() string {
	return azureDiskType
}

func createAzureDisk(i interface{}) VolumeSource {
	azureVolume, ok := i.(*v1.AzureDiskVolumeSource)
	if !ok {
		return &Unimplemented{}
	}
	return &azureDisk{
		AzureDiskVolumeSource: azureVolume,
	}
}

func init() {
	VolumeRegistry[azureDiskType] = createAzureDisk
}
