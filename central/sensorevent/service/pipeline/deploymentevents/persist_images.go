package deploymentevents

import (
	"bitbucket.org/stack-rox/apollo/central/image/datastore"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/images"
)

func newPersistImages(images datastore.DataStore) *persistImagesImpl {
	return &persistImagesImpl{
		images: images,
	}
}

type persistImagesImpl struct {
	images datastore.DataStore
}

func (s *persistImagesImpl) do(deployment *v1.Deployment) {
	for _, i := range images.FromContainers(deployment.GetContainers()).Images() {
		if i.GetName().GetSha() == "" {
			log.Debugf("Skipping persistence of image without sha: %s", i.GetName().GetFullName())
			continue
		}

		if err := s.images.UpdateImage(i); err != nil {
			log.Error(err)
		}
	}
}
