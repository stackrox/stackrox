package manager

import (
	"testing"
	"time"

	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/coalescer"
	"github.com/stackrox/rox/pkg/images/utils"
	"github.com/stackrox/rox/pkg/sizeboundedcache"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type ImageCacheTestSuite struct {
	suite.Suite
	manager *manager
}

func TestImageCacheTestSuite(t *testing.T) {
	suite.Run(t, new(ImageCacheTestSuite))
}

func (s *ImageCacheTestSuite) SetupSuite() {
	cache, err := sizeboundedcache.New(1024*1024, 512*1024, func(k string, v imageCacheEntry) int64 {
		return 1024 // Simple constant size for testing
	})
	require.NoError(s.T(), err)

	nameCache, err := lru.New[string, string](imageNameCacheSize)
	require.NoError(s.T(), err)

	s.manager = &manager{
		imageCache:               cache,
		imageNameToImageCacheKey: nameCache,
		imageNameCacheEnabled:    true,
		imageFetchGroup:          coalescer.New[*storage.Image](),
		imageCacheGen:            newImageGenTracker(),
	}
}

func (s *ImageCacheTestSuite) SetupTest() {
	s.manager.imageCache.Purge()
	s.manager.imageNameToImageCacheKey.Purge()
	s.manager.imageCacheGen.Clear()
	s.manager.state.Store(nil)
}

func createTestState(flattenImageData bool) *state {
	return &state{
		AdmissionControlSettings: &sensor.AdmissionControlSettings{
			FlattenImageData: flattenImageData,
		},
	}
}

func (s *ImageCacheTestSuite) TestGetCachedImage() {
	imageName := &storage.ImageName{
		Registry: "docker.io",
		Remote:   "library/nginx",
		Tag:      "latest",
		FullName: "docker.io/library/nginx:latest",
	}

	tests := []struct {
		name           string
		state          *state
		containerImage *storage.ContainerImage
		cachedImage    *storage.Image
		expectFound    bool
	}{
		{
			name:  "without capability - cache hit with ID",
			state: createTestState(false),
			containerImage: &storage.ContainerImage{
				Id:   "sha256:abc123",
				Name: imageName,
			},
			cachedImage: &storage.Image{
				Id:   "sha256:abc123",
				Name: imageName,
			},
			expectFound: true,
		},
		{
			name:  "without capability - cache miss with ID",
			state: createTestState(false),
			containerImage: &storage.ContainerImage{
				Id:   "sha256:notcached",
				Name: imageName,
			},
			cachedImage: &storage.Image{
				Id:   "sha256:abc123",
				Name: imageName,
			},
			expectFound: false,
		},
		{
			name:  "without capability - no ID returns nil",
			state: createTestState(false),
			containerImage: &storage.ContainerImage{
				Id:   "",
				Name: imageName,
			},
			cachedImage: &storage.Image{
				Id:   "sha256:abc123",
				Name: imageName,
			},
			expectFound: false,
		},
		{
			name:  "with capability - cache hit with UUID V5 key",
			state: createTestState(true),
			containerImage: &storage.ContainerImage{
				Id:   "sha256:abc123",
				Name: imageName,
			},
			cachedImage: &storage.Image{
				Id:   "sha256:abc123",
				Name: imageName,
			},
			expectFound: true,
		},
		{
			name:  "with capability - cache miss with different ID",
			state: createTestState(true),
			containerImage: &storage.ContainerImage{
				Id:   "sha256:notcached",
				Name: imageName,
			},
			cachedImage: &storage.Image{
				Id:   "sha256:abc123",
				Name: imageName,
			},
			expectFound: false,
		},
		{
			name:  "with capability - no ID returns nil",
			state: createTestState(true),
			containerImage: &storage.ContainerImage{
				Id:   "",
				Name: imageName,
			},
			cachedImage: &storage.Image{
				Id:   "sha256:abc123",
				Name: imageName,
			},
			expectFound: false,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			if tt.cachedImage != nil {
				var cacheKey string
				if tt.state.GetFlattenImageData() {
					cacheKey = utils.NewImageV2ID(tt.cachedImage.GetName(), tt.cachedImage.GetId())
				} else {
					cacheKey = tt.cachedImage.GetId()
				}
				s.manager.imageCache.Add(cacheKey, imageCacheEntry{
					Image:     tt.cachedImage,
					timestamp: time.Now(),
				})
			}

			result := s.manager.getCachedImage(tt.containerImage, tt.state, true)

			if tt.expectFound {
				s.NotNil(result)
				s.Equal(tt.cachedImage.GetId(), result.GetId())
			} else {
				s.Nil(result)
			}
		})
	}
}

func (s *ImageCacheTestSuite) TestCacheImage() {
	imageName := &storage.ImageName{
		Registry: "docker.io",
		Remote:   "library/nginx",
		Tag:      "latest",
		FullName: "docker.io/library/nginx:latest",
	}

	tests := []struct {
		name        string
		state       *state
		image       *storage.Image
		shouldCache bool
	}{
		{
			name:  "without capability - do not cache image without ID",
			state: createTestState(false),
			image: &storage.Image{
				Id:   "",
				Name: imageName,
			},
			shouldCache: false,
		},
		{
			name:  "with capability - do not cache image without ID",
			state: createTestState(true),
			image: &storage.Image{
				Id:   "",
				Name: imageName,
			},
			shouldCache: false,
		},
		{
			name:  "without capability - cache image with ID",
			state: createTestState(false),
			image: &storage.Image{
				Id:   "sha256:abc123",
				Name: imageName,
			},
			shouldCache: true,
		},
		{
			name:  "with capability - cache image with UUID V5 key",
			state: createTestState(true),
			image: &storage.Image{
				Id:   "sha256:abc123",
				Name: imageName,
			},
			shouldCache: true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.manager.cacheImage(tt.image, tt.image.GetName().GetFullName(), tt.state)

			if tt.shouldCache {
				var expectedKey string
				if tt.state.GetFlattenImageData() {
					expectedKey = utils.NewImageV2ID(tt.image.GetName(), tt.image.GetId())
				} else {
					expectedKey = tt.image.GetId()
				}
				cachedEntry, found := s.manager.imageCache.Get(expectedKey)
				s.True(found)
				s.Equal(tt.image.GetId(), cachedEntry.Image.GetId())
				s.Equal(tt.image.GetName().GetFullName(), cachedEntry.Image.GetName().GetFullName())
			} else {
				objects, _ := s.manager.imageCache.Stats()
				s.Equal(int64(0), objects)
			}
		})
	}
}

// --- test helpers ---

func (s *ImageCacheTestSuite) addCacheEntry(key string, img *storage.Image) {
	s.manager.imageCache.Add(key, imageCacheEntry{Image: img, timestamp: time.Now()})
}

func (s *ImageCacheTestSuite) addNameMapping(name, key string) {
	s.manager.imageNameToImageCacheKey.Add(name, key)
}

func (s *ImageCacheTestSuite) assertCached(key string, expected bool, msgAndArgs ...interface{}) {
	_, ok := s.manager.imageCache.Get(key)
	s.Equal(expected, ok, msgAndArgs...)
}

func (s *ImageCacheTestSuite) assertNameMapped(name string, expected bool, msgAndArgs ...interface{}) {
	_, ok := s.manager.imageNameToImageCacheKey.Get(name)
	s.Equal(expected, ok, msgAndArgs...)
}

func (s *ImageCacheTestSuite) invalidate(keys ...*central.ImageKey) {
	s.manager.processImageCacheInvalidation(&sensor.AdmCtrlImageCacheInvalidation{ImageKeys: keys})
}

// --- invalidation tests ---

func (s *ImageCacheTestSuite) TestProcessImageInvalidation_RemovesTargetedEntry() {
	s.manager.state.Store(createTestState(false))

	s.addCacheEntry("sha256:nginx", &storage.Image{Id: "sha256:nginx"})
	s.addCacheEntry("sha256:redis", &storage.Image{Id: "sha256:redis"})
	s.addNameMapping("docker.io/library/nginx:1.25", "sha256:nginx")
	s.addNameMapping("docker.io/library/redis:7", "sha256:redis")

	s.invalidate(&central.ImageKey{
		ImageId: "sha256:nginx", ImageFullName: "docker.io/library/nginx:1.25",
	})

	s.assertCached("sha256:nginx", false, "nginx should be removed")
	s.assertNameMapped("docker.io/library/nginx:1.25", false, "nginx name mapping should be removed")
	s.assertCached("sha256:redis", true, "redis should remain")
	s.assertNameMapped("docker.io/library/redis:7", true, "redis name mapping should remain")
}

func (s *ImageCacheTestSuite) TestProcessImageInvalidation_WithFlattenImageData() {
	s.manager.state.Store(createTestState(true))
	v2Key := "v2-uuid-for-nginx"

	s.addCacheEntry(v2Key, &storage.Image{Id: "sha256:nginx"})
	s.addNameMapping("docker.io/library/nginx:1.25", v2Key)

	s.invalidate(&central.ImageKey{
		ImageId: "sha256:nginx", ImageIdV2: v2Key, ImageFullName: "docker.io/library/nginx:1.25",
	})

	s.assertCached(v2Key, false, "V2 cache entry should be removed")
	s.assertNameMapped("docker.io/library/nginx:1.25", false, "name mapping should be removed")
}

func (s *ImageCacheTestSuite) TestProcessImageInvalidation_IncrementsGeneration() {
	s.manager.state.Store(createTestState(false))

	s.invalidate(&central.ImageKey{
		ImageId: "sha256:abc", ImageFullName: "docker.io/library/nginx:1.25",
	})

	s.Equal(uint64(1), s.manager.imageCacheGen.Get("sha256:abc"))
	s.Equal(uint64(1), s.manager.imageCacheGen.Get("docker.io/library/nginx:1.25"))
}

func (s *ImageCacheTestSuite) TestProcessImageInvalidation_MultipleKeys() {
	s.manager.state.Store(createTestState(false))
	for _, id := range []string{"sha256:a", "sha256:b", "sha256:c"} {
		s.addCacheEntry(id, &storage.Image{Id: id})
	}

	s.invalidate(
		&central.ImageKey{ImageId: "sha256:a"},
		&central.ImageKey{ImageId: "sha256:c"},
	)

	s.assertCached("sha256:a", false)
	s.assertCached("sha256:b", true, "sha256:b should not be invalidated")
	s.assertCached("sha256:c", false)
}

// --- generation counter tests ---

func (s *ImageCacheTestSuite) TestGenerationCounter() {
	st := createTestState(false)
	imgName := &storage.ImageName{FullName: "docker.io/library/nginx:1.25"}
	img := &storage.Image{Id: "sha256:abc", Name: imgName}

	tests := []struct {
		name        string
		mutate      func()
		expectCache bool
	}{
		{
			name:        "allows cache when unchanged",
			mutate:      func() {},
			expectCache: true,
		},
		{
			name:        "prevents stale cache after per-key Inc",
			mutate:      func() { s.manager.imageCacheGen.Inc("sha256:abc") },
			expectCache: false,
		},
		{
			name:        "prevents stale cache after CacheVersion change",
			mutate:      func() { s.manager.imageCacheGen.UpdateCacheVersion("new-uuid") },
			expectCache: false,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.manager.imageCache.Purge()
			s.manager.imageCacheGen.Clear()

			gen, cacheVer := s.manager.imageCacheGen.Snapshot("sha256:abc")
			tt.mutate()
			if !s.manager.imageCacheGen.Changed("sha256:abc", gen, cacheVer) {
				s.manager.cacheImage(img, imgName.GetFullName(), st)
			}
			s.assertCached("sha256:abc", tt.expectCache)
		})
	}
}

func (s *ImageCacheTestSuite) TestClearImageCacheGen() {
	s.manager.imageCacheGen.Inc("sha256:a")
	s.manager.imageCacheGen.Inc("sha256:a")
	s.manager.imageCacheGen.Inc("sha256:b")
	s.manager.imageCacheGen.UpdateCacheVersion("v1")

	s.manager.imageCacheGen.Clear()

	s.Equal(uint64(0), s.manager.imageCacheGen.Get("sha256:a"))
	s.Equal(uint64(0), s.manager.imageCacheGen.Get("sha256:b"))
	s.Equal("", s.manager.imageCacheGen.CacheVersion())
}
