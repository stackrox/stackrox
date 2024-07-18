package imagecacheutils

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/expiringcache"
)

type ImageCache expiringcache.Cache[Key, Value]

// Value represents the value stored in image cache
type Value interface {
	// WaitAndGet wait for image scan before returning the image
	WaitAndGet() *storage.Image
	// GetIfDone returns image if scan was done, otherwise it returns nil
	GetIfDone() *storage.Image
}

// Key is the type for keys in image cache to prevent accidental use,
// it should be obtained from image with GetImageCacheKey
type Key string

// CacheKeyProvider represents an interface from which image cache can be generated.
type CacheKeyProvider interface {
	GetId() string
	GetName() *storage.ImageName
}

// GetImageCacheKey generates image cache key from a cache key provider.
func GetImageCacheKey(provider CacheKeyProvider) Key {
	if id := provider.GetId(); id != "" {
		return Key(id)
	}
	return Key(provider.GetName().GetFullName())
}

// CompareImageCacheKey given two CacheKeyProvider, compares if they're equal
func CompareImageCacheKey(a, b CacheKeyProvider) bool {
	if a.GetId() != "" && b.GetId() != "" {
		return a.GetId() == b.GetId()
	}
	return a.GetName().GetFullName() == b.GetName().GetFullName()
}
