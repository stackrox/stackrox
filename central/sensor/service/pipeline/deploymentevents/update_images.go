package deploymentevents

import (
	imageDataStore "github.com/stackrox/rox/central/image/datastore"
	"github.com/stackrox/rox/generated/storage"
)

func newUpdateImages(images imageDataStore.DataStore) *updateImagesImpl {
	return &updateImagesImpl{
		images: images,
	}
}

type updateImagesImpl struct {
	images imageDataStore.DataStore
}

func (s *updateImagesImpl) do(deployment *storage.Deployment) {
	for _, c := range deployment.GetContainers() {
		image := c.GetImage()
		if image.GetId() == "" {
			log.Debugf("Skipping persistence of image without sha: %s", image.GetName().GetFullName())
			continue
		}

		err := s.images.UpsertImage(image)
		if err != nil {
			log.Error(err)
			continue
		}
	}
}
