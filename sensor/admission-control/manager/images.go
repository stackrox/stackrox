package manager

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/images/types"
)

// getImages retrieves a slice of images for the given deployment. Currently, it only returns stub images.
func (m *manager) getImages(deployment *storage.Deployment) []*storage.Image {
	images := make([]*storage.Image, 0, len(deployment.GetContainers()))
	for _, container := range deployment.GetContainers() {
		images = append(images, types.ToImage(container.GetImage()))
	}
	return images
}
