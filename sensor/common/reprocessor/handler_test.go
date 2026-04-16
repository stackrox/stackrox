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

func newTestCache(entries map[cache.Key]string) expiringcache.Cache[cache.Key, cache.Value] {
	c := expiringcache.NewExpiringCache[cache.Key, cache.Value](1 * time.Hour)
	for key, fullName := range entries {
		c.Add(key, &mockImageCacheValue{
			image: &storage.Image{
				Id:   string(key),
				Name: &storage.ImageName{FullName: fullName},
			},
		})
	}
	return c
}

func TestProcessInvalidateImageCache(t *testing.T) {
	cases := []struct {
		name        string
		flatten     bool
		cache       map[cache.Key]string
		imageKeys   []*central.ImageKey
		wantRemoved []cache.Key
		wantKept    []cache.Key
	}{
		{
			name:    "without flatten uses ImageId",
			flatten: false,
			cache: map[cache.Key]string{
				"sha256:abc123": "docker.io/library/nginx:latest",
				"sha256:def456": "quay.io/stackrox/main:4.0.0",
				"redis:alpine":  "redis:alpine",
				"nginx:latest":  "nginx:latest",
			},
			imageKeys: []*central.ImageKey{
				{ImageId: "sha256:abc123", ImageFullName: "docker.io/library/nginx:latest"},
				{ImageId: "sha256:def456", ImageFullName: "quay.io/stackrox/main:4.0.0"},
				{ImageFullName: "redis:alpine"},
			},
			wantRemoved: []cache.Key{"sha256:abc123", "sha256:def456", "redis:alpine"},
			wantKept:    []cache.Key{"nginx:latest"},
		},
		{
			name:    "with flatten uses ImageIdV2",
			flatten: true,
			cache: map[cache.Key]string{
				"uuid-v5-id-1": "docker.io/library/nginx:latest",
				"uuid-v5-id-2": "quay.io/stackrox/main:4.0.0",
				"redis:alpine": "redis:alpine",
				"nginx:latest": "nginx:latest",
			},
			imageKeys: []*central.ImageKey{
				{ImageIdV2: "uuid-v5-id-1", ImageFullName: "docker.io/library/nginx:latest"},
				{ImageIdV2: "uuid-v5-id-2", ImageFullName: "quay.io/stackrox/main:4.0.0"},
				{ImageFullName: "redis:alpine"},
			},
			wantRemoved: []cache.Key{"uuid-v5-id-1", "uuid-v5-id-2", "redis:alpine"},
			wantKept:    []cache.Key{"nginx:latest"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.flatten {
				centralcaps.Set([]centralsensor.CentralCapability{centralsensor.FlattenImageData})
			} else {
				centralcaps.Set([]centralsensor.CentralCapability{})
			}
			t.Cleanup(func() { centralcaps.Set(nil) })

			ctrl := gomock.NewController(t)
			mockAdmCtrl := mocks.NewMockSettingsManager(ctrl)
			mockAdmCtrl.EXPECT().InvalidateImageCache(gomock.Any()).Times(1)

			imgCache := newTestCache(tc.cache)
			handler := &handlerImpl{
				admCtrlSettingsMgr: mockAdmCtrl,
				imageCache:         imgCache,
				stopSig:            concurrency.NewErrorSignal(),
			}

			err := handler.ProcessInvalidateImageCache(&central.InvalidateImageCache{ImageKeys: tc.imageKeys})
			require.NoError(t, err)

			for _, key := range tc.wantRemoved {
				_, found := imgCache.Get(key)
				assert.False(t, found, "key %q should be removed", key)
			}
			for _, key := range tc.wantKept {
				_, found := imgCache.Get(key)
				assert.True(t, found, "key %q should still be in cache", key)
			}
		})
	}
}

func TestProcessRefreshImageCacheTTL(t *testing.T) {
	cases := []struct {
		name      string
		flatten   bool
		cache     map[cache.Key]string
		imageKeys []*central.ImageKey
		wantKept  []cache.Key
	}{
		{
			name:    "without flatten uses ImageId",
			flatten: false,
			cache: map[cache.Key]string{
				"sha256:abc123": "docker.io/library/nginx:latest",
				"sha256:def456": "quay.io/stackrox/main:4.0.0",
			},
			imageKeys: []*central.ImageKey{
				{ImageId: "sha256:abc123", ImageFullName: "docker.io/library/nginx:latest"},
				{ImageId: "sha256:def456", ImageFullName: "quay.io/stackrox/main:4.0.0"},
			},
			wantKept: []cache.Key{"sha256:abc123", "sha256:def456"},
		},
		{
			name:    "with flatten uses ImageIdV2",
			flatten: true,
			cache: map[cache.Key]string{
				"uuid-v5-id-1": "docker.io/library/nginx:latest",
			},
			imageKeys: []*central.ImageKey{
				{ImageIdV2: "uuid-v5-id-1", ImageFullName: "docker.io/library/nginx:latest"},
			},
			wantKept: []cache.Key{"uuid-v5-id-1"},
		},
		{
			name:    "non-existent key does not error",
			flatten: false,
			cache:   map[cache.Key]string{},
			imageKeys: []*central.ImageKey{
				{ImageId: "sha256:nonexistent", ImageFullName: "does-not-exist:latest"},
			},
			wantKept: nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.flatten {
				centralcaps.Set([]centralsensor.CentralCapability{centralsensor.FlattenImageData})
			} else {
				centralcaps.Set([]centralsensor.CentralCapability{})
			}
			t.Cleanup(func() { centralcaps.Set(nil) })

			imgCache := newTestCache(tc.cache)
			handler := &handlerImpl{
				imageCache: imgCache,
				stopSig:    concurrency.NewErrorSignal(),
			}

			err := handler.ProcessRefreshImageCacheTTL(&central.RefreshImageCacheTTL{ImageKeys: tc.imageKeys})
			require.NoError(t, err)

			for _, key := range tc.wantKept {
				_, found := imgCache.Get(key)
				assert.True(t, found, "key %q should still be in cache after TTL refresh", key)
			}
		})
	}
}
