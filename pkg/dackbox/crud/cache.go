package crud

import (
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	log = logging.LoggerForModule()
)

// NewCache creates a new cache
func NewCache() *Cache {
	return &Cache{
		cache: make(map[string]proto.Message),
	}
}

// Cache is a dackbox cache that is lazily populated on upserts
type Cache struct {
	lock  sync.RWMutex
	cache map[string]proto.Message
}

// Exists returns if the key is in the cache
func (c *Cache) Exists(key []byte) bool {
	c.lock.RLock()
	defer c.lock.RUnlock()

	_, ok := c.cache[string(key)]
	return ok
}

// Get returns the cloned message for the passed key if it exists
func (c *Cache) Get(key []byte) (proto.Message, bool) {
	start := time.Now()
	c.lock.RLock()
	log.Infof("Get runlock took: %d ms", time.Since(start))
	defer c.lock.RUnlock()

	if msg, ok := c.cache[string(key)]; ok {
		defer func(start time.Time) {
			log.Infof("Get clone took: %d ms", time.Since(start))
		}(time.Now())
		return proto.Clone(msg), true
	}
	return nil, false
}

// Set populates the cache
func (c *Cache) Set(key []byte, msg proto.Message) {
	start := time.Now()
	c.lock.Lock()
	log.Infof("Set lock took: %d ms", time.Since(start))
	defer c.lock.Unlock()

	defer func(start time.Time) {
		log.Infof("set and clone took: %d ms", time.Since(start))
	}(time.Now())
	c.cache[string(key)] = proto.Clone(msg)
}

// Delete removes an item from the cache
func (c *Cache) Delete(key []byte) {
	start := time.Now()
	c.lock.Lock()
	log.Infof("delete lock took: %d ms", time.Since(start))
	defer c.lock.Unlock()

	defer func(start time.Time) {
		log.Infof("delete key took: %d ms", time.Since(start))
	}(time.Now())
	delete(c.cache, string(key))
}
