package cache

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stackrox/rox/central/image/datastore/internal/store/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/images/types"
	"github.com/stackrox/rox/pkg/images/utils"
	"github.com/stretchr/testify/assert"
)

func TestImageCache(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	baseStore := mocks.NewMockStore(mockCtrl)
	cacheStore := NewCachedStore(baseStore)

	img1 := fixtures.GetImage()
	img1WithoutCVESummaries := utils.StripCVEDescriptions(img1)
	listImg1 := types.ConvertImageToListImage(img1)

	img2 := fixtures.GetImage()
	img2.Id = "id2"
	img2WithoutCVESummaries := utils.StripCVEDescriptions(img2)

	baseStore.EXPECT().ListImage(img1.GetId()).Return(nil, false, nil)
	listImg, exists, err := cacheStore.ListImage(img1.GetId())
	assert.NoError(t, err)
	assert.False(t, exists)
	assert.Nil(t, listImg)

	// Call Get and return img1 as if it already exists in the store
	// This should fill the cache
	baseStore.EXPECT().GetImage(img1.GetId(), false).Return(img1, true, nil)
	img, exists, err := cacheStore.GetImage(img1.GetId(), false)
	assert.NoError(t, err)
	assert.True(t, exists)
	assert.Equal(t, img1, img)

	listImg, exists, err = cacheStore.ListImage(img1.GetId())
	assert.NoError(t, err)
	assert.True(t, exists)
	assert.Equal(t, listImg1, listImg)

	baseStore.EXPECT().Upsert(img1).Return(nil)
	assert.NoError(t, cacheStore.Upsert(img1))
	baseStore.EXPECT().Upsert(img2).Return(nil)
	assert.NoError(t, cacheStore.Upsert(img2))

	// Don't expect this to hit the underlying store
	img, exists, err = cacheStore.GetImage(img1.GetId(), false)
	assert.NoError(t, err)
	assert.True(t, exists)
	assert.Equal(t, img1WithoutCVESummaries, img)

	// Expect withDescription=true to hit the store
	baseStore.EXPECT().GetImage(img1.GetId(), true).Return(img1, true, nil)
	img, exists, err = cacheStore.GetImage(img1.GetId(), true)
	assert.NoError(t, err)
	assert.True(t, exists)
	assert.Equal(t, img1, img)

	images, missing, err := cacheStore.GetImagesBatch([]string{img1.GetId(), img2.GetId()})
	assert.NoError(t, err)
	assert.Empty(t, missing)
	assert.Equal(t, []*storage.Image{img1WithoutCVESummaries, img2WithoutCVESummaries}, images)

	baseStore.EXPECT().GetImage("noid", false)
	images, missing, err = cacheStore.GetImagesBatch([]string{img1.GetId(), "noid", img2.GetId()})
	assert.NoError(t, err)
	assert.Equal(t, []int{1}, missing)
	assert.Equal(t, []*storage.Image{img1WithoutCVESummaries, img2WithoutCVESummaries}, images)

	listImage, exists, err := cacheStore.ListImage(img1.GetId())
	assert.NoError(t, err)
	assert.True(t, exists)
	assert.Equal(t, types.ConvertImageToListImage(img1), listImage)

	img, exists, err = cacheStore.GetImage(img2.GetId(), false)
	assert.NoError(t, err)
	assert.True(t, exists)
	assert.Equal(t, img2WithoutCVESummaries, img)

	baseStore.EXPECT().Delete(img1.GetId()).Return(nil)
	assert.NoError(t, cacheStore.Delete(img1.GetId()))

	// Expect the cache to be hit with a tombstone and the DB will not be hit
	img, exists, err = cacheStore.GetImage(img1.GetId(), false)
	assert.NoError(t, err)
	assert.False(t, exists)
	assert.Nil(t, img)

	// Test acknowledgements
	baseStore.EXPECT().AckKeysIndexed(img1.GetId(), img2.GetId()).Return(nil)
	assert.NoError(t, cacheStore.AckKeysIndexed(img1.GetId(), img2.GetId()))
}
