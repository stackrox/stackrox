package cache

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stackrox/rox/central/image/datastore/internal/store/mocks"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/images/types"
	"github.com/stretchr/testify/assert"
)

func TestImageCache(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	baseStore := mocks.NewMockStore(mockCtrl)
	cacheStore := NewCachedStore(baseStore)

	img1 := fixtures.GetImage()

	img2 := fixtures.GetImage()
	img2.Id = "id2"

	// Call Get and return img1 as if it already exists in the store
	// This should fill the cache
	baseStore.EXPECT().GetImage(img1.GetId()).Return(img1, true, nil)
	img, exists, err := cacheStore.GetImage(img1.GetId())
	assert.NoError(t, err)
	assert.True(t, exists)
	assert.Equal(t, img1, img)

	baseStore.EXPECT().Upsert(img1).Return(nil)
	assert.NoError(t, cacheStore.Upsert(img1))
	baseStore.EXPECT().Upsert(img2).Return(nil)
	assert.NoError(t, cacheStore.Upsert(img2))

	// Don't expect this to hit the underlying store
	img, exists, err = cacheStore.GetImage(img1.GetId())
	assert.NoError(t, err)
	assert.True(t, exists)
	assert.Equal(t, img1, img)

	listImage, exists, err := cacheStore.ListImage(img1.GetId())
	assert.NoError(t, err)
	assert.True(t, exists)
	assert.Equal(t, types.ConvertImageToListImage(img1), listImage)

	img, exists, err = cacheStore.GetImage(img2.GetId())
	assert.NoError(t, err)
	assert.True(t, exists)
	assert.Equal(t, img2, img)

	baseStore.EXPECT().Delete(img1.GetId()).Return(nil)
	assert.NoError(t, cacheStore.Delete(img1.GetId()))

	// Expect the cache to be hit with a tombstone and the DB will not be hit
	img, exists, err = cacheStore.GetImage(img1.GetId())
	assert.NoError(t, err)
	assert.False(t, exists)
	assert.Nil(t, img)

	// Test acknowledgements
	baseStore.EXPECT().AckKeysIndexed(img1.GetId(), img2.GetId()).Return(nil)
	assert.NoError(t, cacheStore.AckKeysIndexed(img1.GetId(), img2.GetId()))
}
