package reprocessor

import (
	"testing"
	"time"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/expiringcache"
	"github.com/stackrox/rox/sensor/common/admissioncontroller/mocks"
	"github.com/stackrox/rox/sensor/common/centralcaps"
	"github.com/stackrox/rox/sensor/common/image/cache"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

type mockImageCacheValue struct {
	image *storage.Image
}

func (m *mockImageCacheValue) WaitAndGet() *storage.Image {
	return m.image
}

func (m *mockImageCacheValue) GetIfDone() *storage.Image {
	return m.image
}

func TestProcessInvalidateImageCache_WithoutFlattenImageData(t *testing.T) {
	centralcaps.Set([]centralsensor.CentralCapability{})

	ctrl := gomock.NewController(t)
	mockAdmCtrlSettingsMgr := mocks.NewMockSettingsManager(ctrl)
	mockAdmCtrlSettingsMgr.EXPECT().FlushCache().Times(1)

	imageCache := expiringcache.NewExpiringCache[cache.Key, cache.Value](1 * time.Hour)

	imageCache.Add(cache.Key("sha256:abc123"), &mockImageCacheValue{
		image: &storage.Image{
			Id:   "sha256:abc123",
			Name: &storage.ImageName{FullName: "docker.io/library/nginx:latest"},
		},
	})
	imageCache.Add(cache.Key("sha256:def456"), &mockImageCacheValue{
		image: &storage.Image{
			Id:   "sha256:def456",
			Name: &storage.ImageName{FullName: "quay.io/stackrox/main:4.0.0"},
		},
	})
	// Images without IDs (cached by full name)
	imageCache.Add(cache.Key("redis:alpine"), &mockImageCacheValue{
		image: &storage.Image{
			Id:   "",
			Name: &storage.ImageName{FullName: "redis:alpine"},
		},
	})
	imageCache.Add(cache.Key("nginx:latest"), &mockImageCacheValue{
		image: &storage.Image{
			Id:   "",
			Name: &storage.ImageName{FullName: "nginx:latest"},
		},
	})

	// Verify images are in cache
	_, found := imageCache.Get(cache.Key("sha256:abc123"))
	require.True(t, found, "Image sha256:abc123 should be in cache")
	_, found = imageCache.Get(cache.Key("sha256:def456"))
	require.True(t, found, "Image sha256:def456 should be in cache")
	_, found = imageCache.Get(cache.Key("redis:alpine"))
	require.True(t, found, "Image redis:alpine should be in cache")
	_, found = imageCache.Get(cache.Key("nginx:latest"))
	require.True(t, found, "Image nginx:latest should be in cache")

	handler := &handlerImpl{
		admCtrlSettingsMgr: mockAdmCtrlSettingsMgr,
		imageCache:         imageCache,
		stopSig:            concurrency.NewErrorSignal(),
	}

	// Create invalidate request (without capability, uses ImageId or full name)
	req := &central.InvalidateImageCache{
		ImageKeys: []*central.InvalidateImageCache_ImageKey{
			{
				ImageId:       "sha256:abc123",
				ImageFullName: "docker.io/library/nginx:latest",
			},
			{
				ImageId:       "sha256:def456",
				ImageFullName: "quay.io/stackrox/main:4.0.0",
			},
			{
				ImageFullName: "redis:alpine",
			},
		},
	}

	err := handler.ProcessInvalidateImageCache(req)
	require.NoError(t, err)

	// Verify specified images are removed from cache
	_, found = imageCache.Get(cache.Key("sha256:abc123"))
	assert.False(t, found, "Image sha256:abc123 should be removed from cache")
	_, found = imageCache.Get(cache.Key("sha256:def456"))
	assert.False(t, found, "Image sha256:def456 should be removed from cache")
	_, found = imageCache.Get(cache.Key("redis:alpine"))
	assert.False(t, found, "Image redis:alpine should be removed from cache (fallback to full name)")

	// Verify image not in request is still in cache
	_, found = imageCache.Get(cache.Key("nginx:latest"))
	assert.True(t, found, "Image nginx:latest should still be in cache")
}

func TestProcessInvalidateImageCache_WithFlattenImageData(t *testing.T) {
	centralcaps.Set([]centralsensor.CentralCapability{centralsensor.FlattenImageData})

	ctrl := gomock.NewController(t)
	mockAdmCtrlSettingsMgr := mocks.NewMockSettingsManager(ctrl)
	mockAdmCtrlSettingsMgr.EXPECT().FlushCache().Times(1)

	imageCache := expiringcache.NewExpiringCache[cache.Key, cache.Value](1 * time.Hour)

	imageCache.Add(cache.Key("uuid-v5-id-1"), &mockImageCacheValue{
		image: &storage.Image{
			Id:   "sha256:abc123",
			Name: &storage.ImageName{FullName: "docker.io/library/nginx:latest"},
		},
	})
	imageCache.Add(cache.Key("uuid-v5-id-2"), &mockImageCacheValue{
		image: &storage.Image{
			Id:   "sha256:def456",
			Name: &storage.ImageName{FullName: "quay.io/stackrox/main:4.0.0"},
		},
	})
	// Images without IDs (cached by full name)
	imageCache.Add(cache.Key("redis:alpine"), &mockImageCacheValue{
		image: &storage.Image{
			Id:   "",
			Name: &storage.ImageName{FullName: "redis:alpine"},
		},
	})
	imageCache.Add(cache.Key("nginx:latest"), &mockImageCacheValue{
		image: &storage.Image{
			Id:   "",
			Name: &storage.ImageName{FullName: "nginx:latest"},
		},
	})

	// Verify images are in cache
	_, found := imageCache.Get(cache.Key("uuid-v5-id-1"))
	require.True(t, found, "Image uuid-v5-id-1 should be in cache")
	_, found = imageCache.Get(cache.Key("uuid-v5-id-2"))
	require.True(t, found, "Image uuid-v5-id-2 should be in cache")
	_, found = imageCache.Get(cache.Key("redis:alpine"))
	require.True(t, found, "Image redis:alpine should be in cache")
	_, found = imageCache.Get(cache.Key("nginx:latest"))
	require.True(t, found, "Image nginx:latest should be in cache")

	handler := &handlerImpl{
		admCtrlSettingsMgr: mockAdmCtrlSettingsMgr,
		imageCache:         imageCache,
		stopSig:            concurrency.NewErrorSignal(),
	}

	// Create invalidate request (with capability, uses ImageIdV2 or full name)
	req := &central.InvalidateImageCache{
		ImageKeys: []*central.InvalidateImageCache_ImageKey{
			{
				ImageIdV2:     "uuid-v5-id-1",
				ImageFullName: "docker.io/library/nginx:latest",
			},
			{
				ImageIdV2:     "uuid-v5-id-2",
				ImageFullName: "quay.io/stackrox/main:4.0.0",
			},
			{
				// No ID, fallback to full name
				ImageFullName: "redis:alpine",
			},
		},
	}

	// Call ProcessInvalidateImageCache
	err := handler.ProcessInvalidateImageCache(req)
	require.NoError(t, err)

	// Verify specified images are removed from cache
	_, found = imageCache.Get(cache.Key("uuid-v5-id-1"))
	assert.False(t, found, "Image uuid-v5-id-1 should be removed from cache")
	_, found = imageCache.Get(cache.Key("uuid-v5-id-2"))
	assert.False(t, found, "Image uuid-v5-id-2 should be removed from cache")
	_, found = imageCache.Get(cache.Key("redis:alpine"))
	assert.False(t, found, "Image redis:alpine should be removed from cache (fallback to full name)")

	// Verify image not in request is still in cache
	_, found = imageCache.Get(cache.Key("nginx:latest"))
	assert.True(t, found, "Image nginx:latest should still be in cache")
}
