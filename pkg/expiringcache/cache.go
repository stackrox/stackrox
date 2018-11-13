package expiringcache

import (
	"time"

	"github.com/hashicorp/golang-lru"
)

type elem struct {
	data   interface{}
	expiry time.Time
}

// Cache implements a cache where the elements expire after a specific time
type Cache interface {
	Add(string, interface{})
	Get(string) interface{}
	GetAll() []interface{}
	Purge()
}

type expiringCacheImpl struct {
	cache  *lru.Cache
	expiry time.Duration
}

// NewExpiringCacheOrPanic returns a new Cache and only panics if the size is negative
// per the lru.Cache spec
func NewExpiringCacheOrPanic(size int, expiry time.Duration) Cache {
	cache, err := lru.New(size)
	if err != nil {
		panic(err)
	}
	return &expiringCacheImpl{
		cache:  cache,
		expiry: expiry,
	}
}

func (e *expiringCacheImpl) Add(k string, i interface{}) {
	if k == "" {
		return
	}
	e.cache.Add(k, &elem{data: i, expiry: time.Now().Add(e.expiry)})
}

// Get takes in a key and returns nil if the item doesn't exist or if the item has expired
func (e *expiringCacheImpl) Get(k string) interface{} {
	el, ok := e.cache.Get(k)
	if !ok || el.(*elem).expiry.Before(time.Now()) {
		return nil
	}
	return el.(*elem).data
}

func (e *expiringCacheImpl) GetAll() []interface{} {
	keys := e.cache.Keys()

	ret := make([]interface{}, 0, len(keys))
	for _, key := range keys {
		el, ok := e.cache.Get(key)
		if !ok || el.(*elem).expiry.Before(time.Now()) {
			continue
		}
		ret = append(ret, el.(*elem).data)
	}
	return ret
}

func (e *expiringCacheImpl) Purge() {
	e.cache.Purge()
}
