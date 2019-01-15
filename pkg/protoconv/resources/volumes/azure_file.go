package volumes

import (
	"k8s.io/api/core/v1"
)

const azureFileType = "AzureFile"

type azureFile struct {
	*v1.AzureFileVolumeSource
}

// Source returns the source of the specific implementation
func (h *azureFile) Source() string {
	return h.ShareName
}

func (h *azureFile) Type() string {
	return azureFileType
}

func createAzureFile(i interface{}) VolumeSource {
	azureVolume, ok := i.(*v1.AzureFileVolumeSource)
	if !ok {
		return &Unimplemented{}
	}
	return &azureFile{
		AzureFileVolumeSource: azureVolume,
	}
}

func init() {
	VolumeRegistry[azureFileType] = createAzureFile
}
