package cache

import (
	"time"

	"github.com/stackrox/rox/pkg/expiringcache"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	metadataCacheOnce sync.Once
	metadataCache     ImageMetadata
)

const imageCacheExpiryDuration = 4 * time.Hour

type ImageMetadata expiringcache.Cache

// ImageMetadataCacheSingleton returns the cache for image metadata
func ImageMetadataCacheSingleton() ImageMetadata {
	metadataCacheOnce.Do(func() {
		metadataCache = expiringcache.NewExpiringCache(imageCacheExpiryDuration, expiringcache.UpdateExpirationOnGets)
	})
	return metadataCache
}
