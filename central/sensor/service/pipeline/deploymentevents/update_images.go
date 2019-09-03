package deploymentevents

import (
	"context"

	imageDataStore "github.com/stackrox/rox/central/image/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/images/types"
)

func newUpdateImages(images imageDataStore.DataStore) *updateImagesImpl {
	return &updateImagesImpl{
		images: images,
	}
}

type updateImagesImpl struct {
	images imageDataStore.DataStore
}

func (s *updateImagesImpl) do(ctx context.Context, deployment *storage.Deployment) {
	for _, c := range deployment.GetContainers() {
		image := c.GetImage()
		if image.GetId() == "" {
			log.Debugf("Skipping persistence of image without sha: %s", image.GetName().GetFullName())
			continue
		}

		fullImage := types.ToImage(c.GetImage())

		err := s.images.UpsertImage(ctx, fullImage)
		if err != nil {
			log.Error(err)
			continue
		}
	}
}
