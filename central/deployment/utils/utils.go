package utils

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/images/utils"
	"github.com/stackrox/rox/pkg/uuid"
)

// GetMaskedDeploymentID returns a deterministic ID value different
// from the input ID in order to hide deployment IDs for deployments
// out of the requester access scope.
func GetMaskedDeploymentID(id string, name string) string {
	return uuid.NewV5FromNonUUIDs(id, name).String()
}

// PopulateContainerImageIDV2s populates the IDV2 field of the container images in the deployment.
func PopulateContainerImageIDV2s(deployment *storage.Deployment) {
	for _, container := range deployment.GetContainers() {
		if container.GetImage().GetIdV2() == "" && container.GetImage().GetId() != "" {
			container.GetImage().IdV2 = utils.NewImageV2ID(container.GetImage().GetName(), container.GetImage().GetId())
		}
	}
}
