package sizeboundedcache

import (
	"math"
	"sync/atomic"

	lru "github.com/hashicorp/golang-lru"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	log = logging.LoggerForModule()
)

// Cache is the interface for a simple size-bounded cache
type Cache interface {
	Add(key, value interface{})
	TestAndSet(key interface{}, value interface{}, pred func(oldValue interface{}, exists bool) bool)
	Get(key interface{}) (interface{}, bool)
	Remove(key interface{})
	RemoveIf(key interface{}, valPred func(interface{}) bool)
	Stats() (objects, size int64)
	Purge()
}

type valueEntry struct {
	totalSize int64
	value     interface{}
}

type sizeBoundedCache struct {
	currSize    int64
	maxSize     int64
	maxItemSize int64
	sizeFunc    func(key, value interface{}) int64

	cacheLock sync.RWMutex
	cache     *lru.Cache
}

// New creates a new cost cache with the passed parameters
func New(maxSize, maxItemSize int64, costFunc func(key, value interface{}) int64) (Cache, error) {
	cache, err := lru.New(math.MaxInt32)
	if err != nil {
		return nil, err
	}
	if maxSize <= 0 {
		return nil, errors.New("max cache size must be greater than 0")
	}
	if maxItemSize <= 0 {
		return nil, errors.New("max item size must be greater than 0")
	}
	if maxSize <= maxItemSize {
		return nil, errors.Errorf("max item size of %d must be less than max cache size of %d", maxItemSize, maxSize)
	}
	if costFunc == nil {
		return nil, errors.New("passed cost func must be non nil")
	}
	return &sizeBoundedCache{
		maxSize:     maxSize,
		maxItemSize: maxItemSize,
		sizeFunc:    costFunc,
		cache:       cache,
	}, nil
}

func (c *sizeBoundedCache) get(key interface{}) (*valueEntry, bool) {
	valueE, ok := c.cache.Get(key)
	if !ok {
		return nil, false
	}
	return valueE.(*valueEntry), true
}

func (c *sizeBoundedCache) Get(key interface{}) (interface{}, bool) {
	valueE, ok := c.get(key)
	if !ok {
		return nil, false
	}
	return valueE.value, true
}

func (c *sizeBoundedCache) Purge() {
	c.cacheLock.Lock()
	defer c.cacheLock.Unlock()

	c.cache.Purge()
	atomic.StoreInt64(&c.currSize, 0)
}

// TestAndSet takes in a key, value and a predicate that must return true for the value to be inserted into the cache
func (c *sizeBoundedCache) TestAndSet(key interface{}, value interface{}, pred func(oldValue interface{}, exists bool) bool) {
	itemSize := c.sizeFunc(key, value)
	if itemSize > c.maxItemSize {
		return
	}
	// This function needs to be atomic so must grab the write lock
	c.cacheLock.Lock()
	defer c.cacheLock.Unlock()

	oldObj, ok := c.cache.Get(key)
	if !pred(oldObj, ok) {
		return
	}

	c.addNoLock(itemSize, key, value)
}

func (c *sizeBoundedCache) addNoLock(itemSize int64, key, value interface{}) {
	var sizeDelta int64
	currValue, ok := c.cache.Get(key)
	if !ok {
		sizeDelta = itemSize
	} else {
		sizeDelta = itemSize - currValue.(*valueEntry).totalSize
	}
	for atomic.LoadInt64(&c.currSize)+sizeDelta > c.maxSize {
		if !c.removeOldestNoLock() {
			log.Error("internal cache error. We should always be able to make room for a valid cache object")
			return
		}
	}
	c.cache.Add(key, &valueEntry{value: value, totalSize: itemSize})
	atomic.AddInt64(&c.currSize, sizeDelta)
}

func (c *sizeBoundedCache) Add(key, value interface{}) {
	itemSize := c.sizeFunc(key, value)
	if itemSize > c.maxItemSize {
		return
	}
	c.cacheLock.Lock()
	defer c.cacheLock.Unlock()

	c.addNoLock(itemSize, key, value)
}

func (c *sizeBoundedCache) removeOldestNoLock() bool {
	_, value, ok := c.cache.RemoveOldest()
	if !ok {
		return false
	}

	atomic.AddInt64(&c.currSize, -value.(*valueEntry).totalSize)

	return true
}

func (c *sizeBoundedCache) Remove(key interface{}) {
	c.RemoveIf(key, nil)
}

func (c *sizeBoundedCache) RemoveIf(key interface{}, valPred func(interface{}) bool) {
	c.cacheLock.Lock()
	defer c.cacheLock.Unlock()

	value, ok := c.get(key)
	if !ok || (valPred != nil && !valPred(value)) {
		return
	}
	c.cache.Remove(key)

	atomic.AddInt64(&c.currSize, -value.totalSize)
}

func (c *sizeBoundedCache) Stats() (objects, size int64) {
	return int64(c.cache.Len()), atomic.LoadInt64(&c.currSize)
}
