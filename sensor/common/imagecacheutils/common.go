package imagecacheutils

import "github.com/stackrox/rox/generated/storage"

// CacheKeyProvider represents an interface from which image cache can be generated.
type CacheKeyProvider interface {
	GetId() string
	GetName() *storage.ImageName
}

// GetImageCacheKey generates image cache key from a cache key provider.
func GetImageCacheKey(provider CacheKeyProvider) string {
	if id := provider.GetId(); id != "" {
		return id
	}
	return provider.GetName().GetFullName()
}

// CompareImageCacheKey given two CacheKeyProvider, compares if they're equal
func CompareImageCacheKey(a, b CacheKeyProvider) bool {
	if a.GetId() != "" && b.GetId() != "" {
		return a.GetId() == b.GetId()
	}
	return a.GetName().GetFullName() == b.GetName().GetFullName()
}
