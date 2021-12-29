package imagecacheutils

import "github.com/stackrox/rox/generated/storage"

// ImageCacheKey represents the key by which images are keyed in image cache.
type ImageCacheKey struct {
	ID, Name string
}

// CacheKeyProvider represents an interface from which image cache can be generated.
type CacheKeyProvider interface {
	GetId() string
	GetName() *storage.ImageName
}

// GetImageCacheKey generates image cache key from a cache key provider.
func GetImageCacheKey(provider CacheKeyProvider) ImageCacheKey {
	return ImageCacheKey{
		ID:   provider.GetId(),
		Name: provider.GetName().GetFullName(),
	}
}
