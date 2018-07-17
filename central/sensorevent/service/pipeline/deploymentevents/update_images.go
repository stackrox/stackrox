package deploymentevents

import (
	imageDataStore "bitbucket.org/stack-rox/apollo/central/image/datastore"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
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
		img, exists, err := s.images.GetImage(c.GetImage().GetName().GetSha())
		if err != nil {
			log.Error(err)
			continue
		}
		if exists {
			c.Image = img
		}
	}
}
