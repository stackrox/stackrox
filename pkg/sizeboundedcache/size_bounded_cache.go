package sizeboundedcache

import (
	"math"
	"sync/atomic"

	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	log = logging.LoggerForModule()
)

// Cache is the interface for a simple size-bounded cache
type Cache[K comparable, V any] interface {
	Add(key K, value V)
	Get(key K) (V, bool)
	Remove(key K)
	RemoveIf(key K, valPred func(V) bool)
	Stats() (objects, size int64)
	Purge()
}

type valueEntry[V any] struct {
	totalSize int64
	value     V
}

type sizeBoundedCache[K comparable, V any] struct {
	currSize    int64
	maxSize     int64
	maxItemSize int64
	sizeFunc    func(key K, value V) int64

	cacheLock sync.RWMutex
	cache     lru.*Cache[K, *valueEntry[V]]
}

// New creates a new cost cache with the passed parameters
func New[K comparable, V any](maxSize, maxItemSize int64, costFunc func(key K, value V) int64) (Cache[K, V], error) {
	cache, err := lru.New[K, *valueEntry[V]](math.MaxInt32)
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
	return &sizeBoundedCache[K, V]{
		maxSize:     maxSize,
		maxItemSize: maxItemSize,
		sizeFunc:    costFunc,
		cache:       cache,
	}, nil
}

func (c *sizeBoundedCache[K, V]) Get(key K) (value V, exists bool) {
	if valueE, ok := c.cache.Get(key); ok {
		return valueE.value, true
	}
	return
}

func (c *sizeBoundedCache[K, V]) Purge() {
	c.cacheLock.Lock()
	defer c.cacheLock.Unlock()

	c.cache.Purge()
	atomic.StoreInt64(&c.currSize, 0)
}

func (c *sizeBoundedCache[K, V]) addNoLock(itemSize int64, key K, value V) {
	var sizeDelta int64
	currValue, ok := c.cache.Get(key)
	if !ok {
		sizeDelta = itemSize
	} else {
		sizeDelta = itemSize - currValue.totalSize
	}
	for atomic.LoadInt64(&c.currSize)+sizeDelta > c.maxSize {
		if !c.removeOldestNoLock() {
			log.Error("internal cache error. We should always be able to make room for a valid cache object")
			return
		}
	}
	c.cache.Add(key, &valueEntry[V]{value: value, totalSize: itemSize})
	atomic.AddInt64(&c.currSize, sizeDelta)
}

func (c *sizeBoundedCache[K, V]) Add(key K, value V) {
	itemSize := c.sizeFunc(key, value)
	if itemSize > c.maxItemSize {
		return
	}
	c.cacheLock.Lock()
	defer c.cacheLock.Unlock()

	c.addNoLock(itemSize, key, value)
}

func (c *sizeBoundedCache[K, V]) removeOldestNoLock() bool {
	_, value, ok := c.cache.RemoveOldest()
	if !ok {
		return false
	}

	atomic.AddInt64(&c.currSize, -value.totalSize)

	return true
}

func (c *sizeBoundedCache[K, V]) Remove(key K) {
	c.RemoveIf(key, nil)
}

func (c *sizeBoundedCache[K, V]) RemoveIf(key K, valPred func(V) bool) {
	c.cacheLock.Lock()
	defer c.cacheLock.Unlock()

	value, ok := c.cache.Get(key)
	if !ok || (valPred != nil && !valPred(value.value)) {
		return
	}
	c.cache.Remove(key)

	atomic.AddInt64(&c.currSize, -value.totalSize)
}

func (c *sizeBoundedCache[K, V]) Stats() (objects, size int64) {
	return int64(c.cache.Len()), atomic.LoadInt64(&c.currSize)
}
