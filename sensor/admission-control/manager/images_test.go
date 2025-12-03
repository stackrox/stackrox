package manager

import (
	"testing"
	"time"

	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/generated/storage"
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

	s.manager = &manager{
		imageCache: cache,
	}
}

func (s *ImageCacheTestSuite) SetupTest() {
	s.manager.imageCache.Purge()
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

			result := s.manager.getCachedImage(tt.containerImage, tt.state)

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
			s.manager.cacheImage(tt.image, tt.state)

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
