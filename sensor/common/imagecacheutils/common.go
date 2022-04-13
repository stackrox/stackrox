package imagecacheutils

import "github.com/stackrox/stackrox/generated/storage"

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
