package enrichment

import (
	"context"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/images/enricher"
	"github.com/stackrox/rox/pkg/images/utils"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	imageV2Mocks "github.com/stackrox/rox/central/imagev2/datastore/mocks"
	enricherMocks "github.com/stackrox/rox/pkg/images/enricher/mocks"
)

func TestEnrichDeploymentV2(t *testing.T) {
	testutils.MustUpdateFeature(t, features.FlattenImageData, true)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockImagesV2 := imageV2Mocks.NewMockDataStore(ctrl)
	mockImageEnricherV2 := enricherMocks.NewMockImageEnricherV2(ctrl)

	e := &enricherImpl{
		images:          nil,
		imagesV2:        mockImagesV2,
		imageEnricher:   nil,
		imageEnricherV2: mockImageEnricherV2,
	}

	ctx := context.Background()
	enrichCtx := enricher.EnrichmentContext{FetchOpt: enricher.ForceRefetch}

	t.Run("empty deployment returns empty images", func(t *testing.T) {
		deployment := &storage.Deployment{
			Containers: []*storage.Container{},
		}
		images, updatedIndices, pending, err := e.EnrichDeploymentV2(ctx, enrichCtx, deployment)
		require.NoError(t, err)
		assert.Empty(t, images)
		assert.Empty(t, updatedIndices)
		assert.False(t, pending)
	})

	t.Run("container with non existing image enriches", func(t *testing.T) {
		idV2 := utils.NewImageV2ID(
			&storage.ImageName{Registry: "reg", Remote: "img", Tag: "latest", FullName: "reg/img:latest"},
			"sha256:abc",
		)
		deployment := &storage.Deployment{
			Containers: []*storage.Container{
				{
					Image: &storage.ContainerImage{
						Name: &storage.ImageName{Registry: "reg", Remote: "img", Tag: "latest", FullName: "reg/img:latest"},
						Id:   "sha256:abc",
						IdV2: idV2,
					},
				},
			},
		}
		useCacheCtx := enricher.EnrichmentContext{FetchOpt: enricher.UseCachesIfPossible}
		mockImagesV2.EXPECT().
			GetImage(gomock.Any(), idV2).
			Return(nil, false, nil)
		mockImageEnricherV2.EXPECT().
			EnrichImage(gomock.Any(), useCacheCtx, gomock.Any()).
			DoAndReturn(func(_ context.Context, _ enricher.EnrichmentContext, img *storage.ImageV2) (enricher.EnrichmentResult, error) {
				assert.Equal(t, idV2, img.GetId())
				assert.Equal(t, "sha256:abc", img.GetDigest())
				return enricher.EnrichmentResult{ImageUpdated: true, ScanResult: enricher.ScanSucceeded}, nil
			})
		images, updatedIndices, pending, err := e.EnrichDeploymentV2(ctx, useCacheCtx, deployment)
		require.NoError(t, err)
		require.Len(t, images, 1)
		assert.Equal(t, []int{0}, updatedIndices)
		assert.False(t, pending)
	})

	t.Run("container with IdV2 fetches from datastore when fetch opts allow", func(t *testing.T) {
		idV2 := utils.NewImageV2ID(
			&storage.ImageName{Registry: "reg", Remote: "img", Tag: "latest", FullName: "reg/img:latest"},
			"sha256:abc",
		)
		cachedImage := &storage.ImageV2{
			Id:     idV2,
			Digest: "sha256:abc",
			Name:   &storage.ImageName{Registry: "reg", Remote: "img", Tag: "latest", FullName: "reg/img:latest"},
		}
		deployment := &storage.Deployment{
			Containers: []*storage.Container{
				{
					Image: &storage.ContainerImage{
						Name: &storage.ImageName{Registry: "reg", Remote: "img", Tag: "latest", FullName: "reg/img:latest"},
						Id:   "sha256:abc",
						IdV2: idV2,
					},
				},
			},
		}
		useCacheCtx := enricher.EnrichmentContext{FetchOpt: enricher.UseCachesIfPossible}
		mockImagesV2.EXPECT().
			GetImage(gomock.Any(), idV2).
			Return(cachedImage, true, nil)
		mockImageEnricherV2.EXPECT().
			EnrichImage(gomock.Any(), useCacheCtx, gomock.Any()).
			Return(enricher.EnrichmentResult{ImageUpdated: false, ScanResult: enricher.ScanSucceeded}, nil)

		images, updatedIndices, pending, err := e.EnrichDeploymentV2(ctx, useCacheCtx, deployment)
		require.NoError(t, err)
		require.Len(t, images, 1)
		assert.Equal(t, idV2, images[0].GetId())
		assert.Empty(t, updatedIndices)
		assert.False(t, pending)
	})

	t.Run("not pullable image is not enriched", func(t *testing.T) {
		idV2 := utils.NewImageV2ID(
			&storage.ImageName{Registry: "reg", Remote: "img", Tag: "latest", FullName: "reg/img:latest"},
			"sha256:abc",
		)
		deployment := &storage.Deployment{
			Containers: []*storage.Container{
				{
					Image: &storage.ContainerImage{
						Name:        &storage.ImageName{Registry: "reg", Remote: "img", Tag: "latest", FullName: "reg/img:latest"},
						Id:          "sha256:abc",
						IdV2:        idV2,
						NotPullable: true,
					},
				},
			},
		}
		// GetImage may or may not be called depending on fetch opts; with ForceRefetch we don't use cache
		// so we go to ToImageV2. With NotPullable, we skip EnrichImage. So no EnrichImage call.
		images, updatedIndices, pending, err := e.EnrichDeploymentV2(ctx, enrichCtx, deployment)
		require.NoError(t, err)
		require.Len(t, images, 1)
		assert.True(t, images[0].GetNotPullable())
		assert.Empty(t, updatedIndices)
		assert.False(t, pending)
	})

	t.Run("enricher ScanTriggered sets pendingEnrichment", func(t *testing.T) {
		deployment := &storage.Deployment{
			Containers: []*storage.Container{
				{
					Image: &storage.ContainerImage{
						Name: &storage.ImageName{Registry: "reg", Remote: "img", Tag: "latest", FullName: "reg/img:latest"},
						Id:   "sha256:abc",
					},
				},
			},
		}
		mockImageEnricherV2.EXPECT().
			EnrichImage(gomock.Any(), enrichCtx, gomock.Any()).
			Return(enricher.EnrichmentResult{ImageUpdated: true, ScanResult: enricher.ScanTriggered}, nil)
		_, _, pending, err := e.EnrichDeploymentV2(ctx, enrichCtx, deployment)
		require.NoError(t, err)
		assert.True(t, pending)
	})
}
