package cache

import (
	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/central/image/datastore/internal/store"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/images/types"
	imageUtils "github.com/stackrox/rox/pkg/images/utils"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/size"
	"github.com/stackrox/rox/pkg/sizeboundedcache"
	"github.com/stackrox/rox/pkg/utils"
)

const (
	maxCachedImageSize = 200 * size.KB // if it's larger than 200KB then we aren't going to cache it
	maxCacheSize       = 200 * size.MB // 200 MB
)

var (
	log = logging.LoggerForModule()
)

type cachedStore struct {
	store store.Store
	cache sizeboundedcache.Cache
}

type imageTombstone struct{}

func sizeFunc(k, v interface{}) int64 {
	if img, ok := v.(*storage.Image); ok {
		return int64(len(k.(string)) + img.Size())
	}
	return int64(len(k.(string)))
}

// NewCachedStore returns an image storage implementation that caches images
func NewCachedStore(store store.Store) store.Store {
	// This is a size based cache, where we use LRU to determine which of the oldest elements should
	// be removed to allow a new element
	cache, err := sizeboundedcache.New(maxCacheSize, maxCachedImageSize, sizeFunc)
	utils.Must(err)

	return &cachedStore{
		store: store,
		cache: cache,
	}
}

func (c *cachedStore) ListImage(sha string) (*storage.ListImage, bool, error) {
	img, hadEntry, err := c.getCachedImage(sha)
	if err != nil {
		return nil, false, err
	}
	if hadEntry {
		if img == nil {
			return nil, false, nil
		}
		return types.ConvertImageToListImage(img), true, nil
	}
	return c.store.ListImage(sha)
}

func (c *cachedStore) GetImages() ([]*storage.Image, error) {
	images, err := c.store.GetImages()
	if err != nil {
		return nil, err
	}
	for _, image := range images {
		c.testAndSetCacheEntry(image)
	}
	return images, nil
}

func (c *cachedStore) CountImages() (int, error) {
	return c.store.CountImages()
}

func (c *cachedStore) getCachedImage(sha string) (*storage.Image, bool, error) {
	entry, ok := c.cache.Get(sha)
	if !ok {
		return nil, false, nil
	}
	if _, ok := entry.(*imageTombstone); ok {
		return nil, true, nil
	}
	return proto.Clone(entry.(*storage.Image)).(*storage.Image), true, nil
}

func (c *cachedStore) testAndSetCacheEntry(image *storage.Image) {
	cachedImage := imageUtils.StripCVEDescriptions(image)
	c.cache.TestAndSet(image.GetId(), cachedImage, func(_ interface{}, exists bool) bool {
		return !exists
	})
	c.updateStats()
}

func (c *cachedStore) GetImage(sha string, withCVESummaries bool) (*storage.Image, bool, error) {
	if !withCVESummaries {
		img, entryExists, err := c.getCachedImage(sha)
		if err != nil {
			return nil, false, err
		}
		if entryExists {
			imageStoreCacheHits.Inc()
			return img, img != nil, nil
		}
	}

	imageStoreCacheMisses.Inc()
	image, exists, err := c.store.GetImage(sha, withCVESummaries)
	if err != nil || !exists {
		return nil, exists, err
	}

	c.testAndSetCacheEntry(image)
	return image, true, nil
}

func (c *cachedStore) GetImagesBatch(shas []string) ([]*storage.Image, []int, error) {
	var images []*storage.Image
	var missingIndices []int
	for i, sha := range shas {
		img, hadEntry, err := c.getCachedImage(sha)
		if err != nil {
			return nil, nil, err
		}
		if hadEntry {
			imageStoreCacheHits.Inc()
			// Tombstone entry existed
			if img == nil {
				missingIndices = append(missingIndices, i)
				continue
			}
			images = append(images, img)
			continue
		}
		imageStoreCacheMisses.Inc()

		img, exists, err := c.store.GetImage(sha, false)
		if err != nil {
			return nil, nil, err
		}
		if !exists {
			missingIndices = append(missingIndices, i)
			continue
		}

		// Add th image to the cache
		c.testAndSetCacheEntry(img)
		images = append(images, img)
	}
	return images, missingIndices, nil
}

func (c *cachedStore) Exists(id string) (bool, error) {
	if _, ok := c.cache.Get(id); ok {
		imageStoreCacheHits.Inc()
		return true, nil
	}
	imageStoreCacheMisses.Inc()
	return c.store.Exists(id)
}

func (c *cachedStore) Upsert(image *storage.Image) error {
	defer c.updateStats()

	if err := c.store.Upsert(image); err != nil {
		return err
	}
	c.cache.Add(image.GetId(), imageUtils.StripCVEDescriptions(image))
	return nil
}

func (c *cachedStore) Delete(id string) error {
	defer c.updateStats()

	if err := c.store.Delete(id); err != nil {
		return err
	}
	c.cache.Add(id, &imageTombstone{})
	return nil
}

func (c *cachedStore) updateStats() {
	objects, size := c.cache.Stats()
	imageStoreCacheObjects.Set(float64(objects))
	imageStoreCacheSize.Set(float64(size))
}

func (c *cachedStore) AckKeysIndexed(keys ...string) error {
	return c.store.AckKeysIndexed(keys...)
}

func (c *cachedStore) GetKeysToIndex() ([]string, error) {
	return c.store.GetKeysToIndex()
}
