package pipeline

import (
	"bitbucket.org/stack-rox/apollo/central/datastore"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/images"
)

func newPersistImages(images datastore.ImageDataStore) *persistImagesImpl {
	return &persistImagesImpl{
		images: images,
	}
}

type persistImagesImpl struct {
	images datastore.ImageDataStore
}

func (s *persistImagesImpl) do(event *v1.DeploymentEvent) {
	for _, i := range images.FromContainers(event.GetDeployment().GetContainers()).Images() {
		if i.GetName().GetSha() == "" {
			log.Debugf("Skipping persistence of image without sha: %s", i.GetName().GetFullName())
			continue
		}

		if err := s.images.UpdateImage(i); err != nil {
			log.Error(err)
		}
	}
}
