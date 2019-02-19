package roxmetadata

import (
	"github.com/stackrox/rox/generated/storage"
)

// Metadata keeps a registry of relevant StackRox-related metadata.
type Metadata interface {
	AddDeployment(*storage.Deployment)
	GetSensorImage() string
}

type metadataImpl struct {
	sensorImage string
}

func (m *metadataImpl) GetSensorImage() string {
	return m.sensorImage
}

func (m *metadataImpl) AddDeployment(deployment *storage.Deployment) {
	if deployment.GetNamespace() == "stackrox" && deployment.GetName() == "sensor" {
		for _, container := range deployment.GetContainers() {
			if container.GetName() == "sensor" {
				if fullName := container.GetImage().GetName().GetFullName(); fullName != "" {
					m.sensorImage = fullName
					return
				}
			}
		}
	}
}

// New returns a new ready-to-use metadata object.
func New() Metadata {
	return &metadataImpl{}
}
