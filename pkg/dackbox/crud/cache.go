package crud

import (
	"github.com/gogo/protobuf/proto"
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
	c.lock.RLock()
	msg, ok := c.cache[string(key)]
	c.lock.RUnlock()
	if ok {
		// This is thread-safe because the cache is the owner of the msg
		// and will not modify it effectively making it read-only
		return proto.Clone(msg), true
	}
	return nil, false
}

// Set populates the cache
func (c *Cache) Set(key []byte, msg proto.Message) {
	cloned := proto.Clone(msg)

	c.lock.Lock()
	defer c.lock.Unlock()

	c.cache[string(key)] = cloned
}

// Delete removes an item from the cache
func (c *Cache) Delete(key []byte) {
	c.lock.Lock()
	defer c.lock.Unlock()

	delete(c.cache, string(key))
}
