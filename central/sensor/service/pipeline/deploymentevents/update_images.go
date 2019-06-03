package deploymentevents

import (
	"context"

	imageDataStore "github.com/stackrox/rox/central/image/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/images/types"
	"github.com/stackrox/rox/pkg/sac"
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
	clusterNSScope := sac.ClusterNSScopeStringFromObject(deployment)

	for _, c := range deployment.GetContainers() {
		image := c.GetImage()
		if image.GetId() == "" {
			log.Debugf("Skipping persistence of image without sha: %s", image.GetName().GetFullName())
			continue
		}

		fullImage := types.ToImage(c.GetImage())

		if fullImage.ClusternsScopes == nil {
			fullImage.ClusternsScopes = make(map[string]string)
		}
		fullImage.ClusternsScopes[deployment.GetId()] = clusterNSScope

		err := s.images.UpsertImage(context.TODO(), fullImage)
		if err != nil {
			log.Error(err)
			continue
		}
	}
}
