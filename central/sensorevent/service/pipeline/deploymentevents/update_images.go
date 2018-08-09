package deploymentevents

import (
	imageDataStore "github.com/stackrox/rox/central/image/datastore"
	"github.com/stackrox/rox/generated/api/v1"
)

func newUpdateImages(images imageDataStore.DataStore) *updateImagesImpl {
	return &updateImagesImpl{
		images: images,
	}
}

type updateImagesImpl struct {
	images imageDataStore.DataStore
}

func (s *updateImagesImpl) do(deployment *v1.Deployment) {
	for _, c := range deployment.GetContainers() {
		image := c.GetImage()
		if image.GetName().GetSha() == "" {
			log.Debugf("Skipping persistence of image without sha: %s", image.GetName().GetFullName())
			continue
		}

		err := s.images.UpsertDedupeImage(image)
		if err != nil {
			log.Error(err)
			continue
		}
	}
}
