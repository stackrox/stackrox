package cache

import (
	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/central/pod/store"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/size"
	"github.com/stackrox/rox/pkg/sizeboundedcache"
	"github.com/stackrox/rox/pkg/utils"
)

const (
	maxCachedPodSize = 5 * size.KB // if it's larger than 5KB, then we aren't going to cache it
	maxCacheSize     = 20 * size.MB
)

var (
	log = logging.LoggerForModule()
)

type podTombstone struct{}

func sizeFunc(k, v interface{}) int64 {
	if pod, ok := v.(*storage.Pod); ok {
		return int64(len(k.(string)) + pod.Size())
	}
	return int64(len(k.(string)))
}

// NewCachedStore returns an deployment store implementation that caches pods
func NewCachedStore(store store.Store) store.Store {
	// This is a size based cache, where we use LRU to determine which of the oldest elements should
	// be removed to allow a new element
	cache, err := sizeboundedcache.New(maxCacheSize, maxCachedPodSize, sizeFunc)
	utils.Must(err)

	return &cachedStore{
		store: store,
		cache: cache,
	}
}

// This cached store implementation relies on the usage of the Pod store,
// so this may not be easily portable to other sections of the code.
type cachedStore struct {
	store store.Store
	cache sizeboundedcache.Cache
}

func (c *cachedStore) testAndSetCacheEntry(id string, pod *storage.Pod) {
	c.cache.TestAndSet(id, pod, func(_ interface{}, exists bool) bool {
		return !exists
	})
	c.updateStats()
}

func (c *cachedStore) getCachedPod(id string) (*storage.Pod, bool, error) {
	entry, ok := c.cache.Get(id)
	if !ok {
		return nil, false, nil
	}
	if _, ok := entry.(*podTombstone); ok {
		return nil, true, nil
	}
	return proto.Clone(entry.(*storage.Pod)).(*storage.Pod), true, nil
}

func (c *cachedStore) Get(id string) (*storage.Pod, bool, error) {
	pod, entryExists, err := c.getCachedPod(id)
	if err != nil {
		return nil, false, err
	}
	if entryExists {
		podStoreCacheHits.Inc()
		// if entry is a tombstone entry, return that the pod doesn't exist
		return pod, pod != nil, nil
	}

	podStoreCacheMisses.Inc()
	pod, exists, err := c.store.Get(id)
	if err != nil || !exists {
		return nil, exists, err
	}

	c.testAndSetCacheEntry(id, pod)
	return pod, true, nil
}

func (c *cachedStore) GetMany(ids []string) ([]*storage.Pod, []int, error) {
	var pods []*storage.Pod
	var missingIndices []int
	for i, id := range ids {
		pod, hadEntry, err := c.getCachedPod(id)
		if err != nil {
			return nil, nil, err
		}
		if hadEntry {
			podStoreCacheHits.Inc()
			// Tombstone entry existed
			if pod == nil {
				missingIndices = append(missingIndices, i)
			} else {
				pods = append(pods, pod)
			}
			continue
		}
		podStoreCacheMisses.Inc()

		pod, exists, err := c.store.Get(id)
		if err != nil {
			return nil, nil, err
		}
		if !exists {
			missingIndices = append(missingIndices, i)
			continue
		}

		// Add the pod to the cache
		c.testAndSetCacheEntry(id, pod)
		pods = append(pods, pod)
	}
	return pods, missingIndices, nil
}

func (c *cachedStore) Upsert(pod *storage.Pod) error {
	if err := c.store.Upsert(pod); err != nil {
		return err
	}

	c.cache.Add(pod.GetId(), pod)
	c.updateStats()
	return nil
}

func (c *cachedStore) Delete(id string) error {
	if err := c.store.Delete(id); err != nil {
		return err
	}
	c.cache.Add(id, &podTombstone{})
	c.updateStats()
	return nil
}

func (c *cachedStore) updateStats() {
	objects, size := c.cache.Stats()
	podStoreCacheObjects.Set(float64(objects))
	podStoreCacheSize.Set(float64(size))
}

func (c *cachedStore) AckKeysIndexed(keys ...string) error {
	return c.store.AckKeysIndexed(keys...)
}

func (c *cachedStore) GetKeysToIndex() ([]string, error) {
	return c.store.GetKeysToIndex()
}

func (c *cachedStore) GetKeys() ([]string, error) {
	return c.store.GetKeys()
}
